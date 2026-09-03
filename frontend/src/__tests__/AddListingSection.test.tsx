import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import userEvent from "@testing-library/user-event";

import { AddListingSection } from "../components/forms/AddListingSection";
import { useCategories, categoryNames, useLocalizedCategoryNames } from "../api/categories";
import { useCreateListing, useUploadListingImage } from "../api/listings";
import type { Category } from "../api/types";
import { makeListing } from "../test/factories";

vi.mock("../api/listings", () => ({
  useCreateListing: vi.fn(),
  useUploadListingImage: vi.fn(),
}));

const createListing = vi.fn();
const uploadImage = vi.fn();
const navigate = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigate };
});

vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn(), useLocalizedCategoryNames: vi.fn() };
});

const TREE: Category[] = [
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

beforeEach(() => {
  createListing.mockReset().mockResolvedValue(makeListing({ id: "new-listing" }));
  uploadImage
    .mockReset()
    .mockResolvedValue({ id: "img-1", url: "/uploads/img-1.jpg", position: 0 });
  navigate.mockReset();
  vi.mocked(useCreateListing).mockReturnValue({
    mutateAsync: createListing,
    isPending: false,
  } as unknown as ReturnType<typeof useCreateListing>);
  vi.mocked(useUploadListingImage).mockReturnValue({
    mutateAsync: uploadImage,
    isPending: false,
  } as unknown as ReturnType<typeof useUploadListingImage>);
});

function mockCategories(data: Category[] | undefined) {
  vi.mocked(useCategories).mockReturnValue({
    data,
    isPending: data === undefined,
    isError: false,
  } as ReturnType<typeof useCategories>);
  // CategorySelect resolves option labels through the localized hook; feed it
  // names derived from whatever data this render is pretending to have.
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(categoryNames(data ?? []));
}

// The section posts a listing and navigates on success, so it needs a query
// client and a router in scope.
function renderSection() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <AddListingSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AddListingSection", () => {
  test("accepts a category that only became valid after the list arrived", async () => {
    mockCategories(undefined);
    const { rerender } = renderSection();

    expect(screen.getByRole("combobox")).toBeDisabled();

    mockCategories(TREE);
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <AddListingSection />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const select = screen.getByRole("combobox");
    await waitFor(() => expect(select).toBeEnabled());

    await userEvent.selectOptions(select, "chanterelles");
    await userEvent.tab();

    await waitFor(() => {
      expect(screen.queryByText("Choose a category from the list")).not.toBeInTheDocument();
    });
    expect(select).toHaveValue("chanterelles");
  });

  // Fills the form well enough for zod to allow submitting.
  async function fillAndSubmit({ withPhoto = false } = {}) {
    mockCategories(TREE);
    renderSection();

    if (withPhoto) {
      const input = document.querySelector('input[type="file"]');
      fireEvent.change(input as HTMLInputElement, {
        target: { files: [new File(["x"], "photo.png", { type: "image/png" })] },
      });
    }

    // Only the price/quantity/unit fields carry a <label>; the rest are
    // headed by an <h2>, so these go by the input's id.
    const field = (name: string) => document.getElementById(name) as HTMLElement;
    await userEvent.type(field("title"), "Chanterelles");
    await userEvent.selectOptions(screen.getByRole("combobox"), "chanterelles");
    await userEvent.type(field("price"), "18");
    await userEvent.type(field("quantity"), "4");
    await userEvent.type(field("unit"), "kg");
    await userEvent.tab();

    const submit = await screen.findByRole("button", { name: "Save" });
    await waitFor(() => expect(submit).toBeEnabled());
    await userEvent.click(submit);
    return submit;
  }

  test("posts the listing and lands on it", async () => {
    await fillAndSubmit();

    await waitFor(() => expect(createListing).toHaveBeenCalled());
    expect(createListing.mock.calls[0][0]).toMatchObject({
      title: "Chanterelles",
      category: "chanterelles",
      unit: "kg",
    });
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/listings/new-listing"));
  });

  test("a refused listing keeps the form and shows why", async () => {
    createListing.mockRejectedValue({ status: 400, message: "Title is too long" });

    await fillAndSubmit();

    expect(await screen.findByRole("alert")).toHaveTextContent("Title is too long");
    expect(navigate).not.toHaveBeenCalled();
  });

  // The listing exists by the time a photo fails, so "couldn't save" would be a
  // lie that sends someone back to make a duplicate. It says what happened and
  // links to what was created.
  test("a listing that saved with a failed photo says so and does not navigate", async () => {
    uploadImage.mockRejectedValue({ status: 413, message: "Image is too large" });

    await fillAndSubmit({ withPhoto: true });

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Listing created, but 1 photo couldn't be uploaded.",
    );
    expect(screen.getByText("This photo is too large to upload.")).toBeInTheDocument();
    // A 413 is the same answer every time - offering a retry would waste a click.
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View the listing" })).toHaveAttribute(
      "href",
      "/listings/new-listing",
    );
    expect(navigate).not.toHaveBeenCalled();

    // The listing exists, so the form must stop offering to create another -
    // the message explains what happened, but a live button still invites the
    // duplicate this branch is here to avoid.
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(createListing).toHaveBeenCalledTimes(1);
  });

  test("a photo lost to the network can be retried, and the last one navigates", async () => {
    uploadImage.mockRejectedValueOnce({ status: 0, message: "Could not reach the server" });

    await fillAndSubmit({ withPhoto: true });

    await screen.findByText("Could not reach the server");
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/listings/new-listing"));
    expect(createListing).toHaveBeenCalledTimes(1);
  });
});
