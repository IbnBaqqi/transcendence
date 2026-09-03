import { act, renderHook } from "@testing-library/react";

import { useImageGallery } from "../hooks/useImageGallery";
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

describe("useImageGallery error messages", () => {
  beforeEach(async () => {
    urlCtor.createObjectURL = vi.fn(() => "blob:mock");
    urlCtor.revokeObjectURL = vi.fn();
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("accepts a valid image", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    act(() => result.current.addFiles([makeFile("a.jpg", "image/jpeg")]));

    expect(result.current.images).toHaveLength(1);
    expect(onError).not.toHaveBeenCalled();
  });

  test("rejects an unsupported type with a localized message", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    act(() => result.current.addFiles([makeFile("notes.txt", "text/plain")]));

    expect(onError).toHaveBeenCalledWith(
      '"notes.txt" isn\'t a supported image type (use JPEG, PNG or WebP).',
    );
    expect(result.current.images).toHaveLength(0);
  });

  test("rejects an oversized file with its size interpolated", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ onError }));

    act(() => result.current.addFiles([makeFile("big.jpg", "image/jpeg", 6 * 1024 * 1024)]));

    expect(onError).toHaveBeenCalledWith('"big.jpg" is larger than 5MB.');
    expect(result.current.images).toHaveLength(0);
  });

  test("uses the singular for a limit of one", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ maxFiles: 1, onError }));

    act(() => result.current.addFiles([makeFile("a.jpg", "image/jpeg")]));
    act(() => result.current.addFiles([makeFile("b.jpg", "image/jpeg")]));

    expect(result.current.images).toHaveLength(1);
    expect(onError).toHaveBeenCalledWith("You can only add up to 1 image.");
  });

  test("uses the plural for a limit above one and adds no more than the limit", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageGallery({ maxFiles: 3, onError }));

    act(() =>
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
    act(() => result.current.addFiles([makeFile("notes.txt", "text/plain")]));

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

  test("moves an image and clamps at the ends", () => {
    const { result } = renderHook(() => useImageGallery());
    act(() =>
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
