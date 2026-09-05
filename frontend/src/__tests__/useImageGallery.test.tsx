import { act, renderHook } from "@testing-library/react";

import { useImageGallery } from "../hooks/useImageGallery";
import { shrinkImage } from "../lib/shrinkImage";
import i18next from "../i18n";

function makeFile(name: string, type: string, bytes = 16) {
  return new File([new ArrayBuffer(bytes)], name, { type });
}

// jsdom ships no blob URL support, so the two methods the hook calls are
// absent. Cast to an optional-typed view so the delete below type-checks.
const urlCtor = URL as unknown as {
  createObjectURL?: (obj: Blob | MediaSource) => string;
  revokeObjectURL?: (url: string) => void;
};

afterAll(() => {
  // Restore (i.e. remove) the stubs from beforeEach so they don't leak into
  // any other test file sharing this worker.
  delete urlCtor.createObjectURL;
  delete urlCtor.revokeObjectURL;
});

// The canvas work itself has no jsdom to run in, so shrinkImage is mocked and
// what is tested here is the contract around it: the order of the checks, and
// that the shrunk file is the one kept.
vi.mock("../lib/shrinkImage", () => ({
  shrinkImage: vi.fn((file: File) => Promise.resolve(file)),
}));

// addFiles awaits shrinkImage now, so every call here is awaited too.
describe("useImageGallery error messages", () => {
  beforeEach(async () => {
    urlCtor.createObjectURL = vi.fn(() => "blob:mock");
    urlCtor.revokeObjectURL = vi.fn();
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("accepts a valid image", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    await act(async () => result.current.addFiles([makeFile("a.jpg", "image/jpeg")]));

    expect(result.current.images).toHaveLength(1);
    expect(onError).not.toHaveBeenCalled();
  });

  test("rejects an unsupported type with a localized message", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    await act(async () => result.current.addFiles([makeFile("notes.txt", "text/plain")]));

    expect(onError).toHaveBeenCalledWith(
      '"notes.txt" isn\'t a supported image type (use JPEG, PNG or WebP).',
    );
    expect(result.current.images).toHaveLength(0);
  });

  // Awaiting the shrink yields, and imagesRef only catches up after setImages
  // commits - so two picks in quick succession read the same budget. Real on
  // the avatar picker, where maxFiles is 1 and shrinking a 6MB photo is not
  // instant.
  test("holds the cap when a second pick lands mid-shrink", async () => {
    let release!: () => void;
    const gate = new Promise<void>((resolve) => (release = resolve));
    vi.mocked(shrinkImage).mockImplementation(async (file: File) => {
      await gate;
      return file;
    });
    const { result } = renderHook(() => useImageGallery({ maxFiles: 1 }));

    await act(async () => {
      const first = result.current.addFiles([makeFile("a.jpg", "image/jpeg")]);
      const second = result.current.addFiles([makeFile("b.jpg", "image/jpeg")]);
      release();
      await Promise.all([first, second]);
    });

    expect(result.current.images).toHaveLength(1);
  });

  // The cap is measured after shrinking, which is the whole point: a phone
  // photo over the limit becomes a file under it instead of a refusal.
  test("measures the cap against the shrunk file, not the original", async () => {
    const onError = vi.fn();
    const small = makeFile("shrunk.jpg", "image/jpeg", 300 * 1024);
    vi.mocked(shrinkImage).mockResolvedValueOnce(small);
    const { result } = renderHook(() => useImageGallery({ onError }));

    await act(async () =>
      result.current.addFiles([makeFile("huge.jpg", "image/jpeg", 8 * 1024 * 1024)]),
    );

    expect(result.current.images).toHaveLength(1);
    expect(result.current.images[0].file).toBe(small);
    expect(onError).not.toHaveBeenCalled();
  });

  // The other order would let a converted file in through the back door: the
  // accepted list is the backend's, and shrinking outputs JPEG.
  test("checks the type before shrinking, so conversion cannot bypass the list", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    await act(async () => result.current.addFiles([makeFile("photo.heic", "image/heic")]));

    expect(shrinkImage).not.toHaveBeenCalled();
    expect(result.current.images).toHaveLength(0);
    expect(onError).toHaveBeenCalled();
  });

  test("rejects an oversized file with its size interpolated", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    await act(async () =>
      result.current.addFiles([makeFile("big.jpg", "image/jpeg", 6 * 1024 * 1024)]),
    );

    expect(onError).toHaveBeenCalledWith('"big.jpg" is larger than 5MB.');
    expect(result.current.images).toHaveLength(0);
  });

  test("uses the singular for a limit of one", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ maxFiles: 1, onError }));

    await act(async () => result.current.addFiles([makeFile("a.jpg", "image/jpeg")]));
    await act(async () => result.current.addFiles([makeFile("b.jpg", "image/jpeg")]));

    expect(result.current.images).toHaveLength(1);
    expect(onError).toHaveBeenCalledWith("You can only add up to 1 image.");
  });

  test("uses the plural for a limit above one and adds no more than the limit", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ maxFiles: 3, onError }));

    await act(async () =>
      result.current.addFiles([
        makeFile("a.jpg", "image/jpeg"),
        makeFile("b.jpg", "image/jpeg"),
        makeFile("c.jpg", "image/jpeg"),
        makeFile("d.jpg", "image/jpeg"),
        makeFile("e.jpg", "image/jpeg"),
      ]),
    );

    expect(result.current.images).toHaveLength(3);
    expect(onError).toHaveBeenCalledWith("You can only add up to 3 images.");
  });

  test("messages follow the active language", async () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    await i18next.changeLanguage("fi");
    await act(async () => result.current.addFiles([makeFile("notes.txt", "text/plain")]));

    expect(onError).toHaveBeenCalledWith(
      '"notes.txt" ei ole tuettu kuvatyyppi (käytä JPEG-, PNG- tai WebP-kuvia).',
    );
  });
});

describe("useImageGallery ordering", () => {
  beforeEach(() => {
    urlCtor.createObjectURL = vi.fn(() => "blob:mock");
    urlCtor.revokeObjectURL = vi.fn();
  });

  test("moves an image and clamps at the ends", async () => {
    const { result } = renderHook(() => useImageGallery());
    await act(async () =>
      result.current.addFiles([
        makeFile("a.jpg", "image/jpeg"),
        makeFile("b.jpg", "image/jpeg"),
        makeFile("c.jpg", "image/jpeg"),
      ]),
    );

    act(() => result.current.moveImage(result.current.images[2].id, -1));
    expect(result.current.images.map((i) => i.file.name)).toEqual(["a.jpg", "c.jpg", "b.jpg"]);

    // The first image has nowhere earlier to go.
    act(() => result.current.moveImage(result.current.images[0].id, -1));
    expect(result.current.images.map((i) => i.file.name)).toEqual(["a.jpg", "c.jpg", "b.jpg"]);
  });
});
