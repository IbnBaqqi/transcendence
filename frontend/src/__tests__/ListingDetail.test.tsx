import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import ListingDetail from "../pages/ListingDetail";
import { useListing } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { makeListing } from "../test/factories";
import type { Listing, ListingImage } from "../api/types";

// The four sections under the page each fetch their own data and are not what
// this page is being tested for.
vi.mock("../api/listings", () => ({ useListing: vi.fn() }));
vi.mock("../api/categories", () => ({ useLocalizedCategoryNames: vi.fn() }));
vi.mock("../components/forms/ReserveListingSection", () => ({ ReserveListingSection: () => null }));
vi.mock("../components/forms/StartConversationSection", () => ({
  StartConversationSection: () => null,
}));
vi.mock("../components/forms/ReportListingSection", () => ({ ReportListingSection: () => null }));
vi.mock("../components/objects/SellerFollowSection", () => ({ SellerFollowSection: () => null }));

beforeEach(() => {
  vi.mocked(useLocalizedCategoryNames).mockReturnValue((slug: string) => `name:${slug}`);
});

const photos: ListingImage[] = [
  { id: "i1", url: "/uploads/first.jpg", position: 0 },
  { id: "i2", url: "/uploads/second.jpg", position: 1 },
];

function renderDetail(listing: Listing | undefined, state: Record<string, unknown> = {}) {
  vi.mocked(useListing).mockReturnValue({
    data: listing,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof useListing>);

  return render(
    <MemoryRouter initialEntries={["/listings/l1"]}>
      <Routes>
        <Route path="/listings/:id" element={<ListingDetail />} />
      </Routes>
    </MemoryRouter>,
  );
}

// It rendered "Listing {id}" and a "not built yet" line until #265.
test("names the listing rather than its id", () => {
  renderDetail(makeListing({ title: "Golden Chanterelles" }));

  expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Golden Chanterelles");
  expect(screen.getByText(/18.00/)).toBeInTheDocument();
  expect(screen.getByText("Freshly foraged this morning.")).toBeInTheDocument();
});

// The photo is the product here, so it gets a real description - unlike the
// thumbnails, which only change which one is large.
test("shows the first photo, described, with the rest as thumbnails", () => {
  const { container } = renderDetail(makeListing({ images: photos }));

  const main = container.querySelector("img");
  expect(main).toHaveAttribute("src", "/uploads/first.jpg");
  expect(main).toHaveAttribute("alt", "Golden Chanterelles");
  expect(screen.getByRole("button", { name: "Show photo 2" })).toBeInTheDocument();
});

test("swaps the main photo when a thumbnail is chosen", async () => {
  const user = userEvent.setup();
  const { container } = renderDetail(makeListing({ images: photos }));

  await user.click(screen.getByRole("button", { name: "Show photo 2" }));

  expect(container.querySelector("img")).toHaveAttribute("src", "/uploads/second.jpg");
});

// A strip that cannot change anything is furniture.
test("offers no thumbnail strip for a single photo", () => {
  renderDetail(makeListing({ images: [photos[0]] }));

  expect(screen.queryByRole("button", { name: /Show photo/ })).not.toBeInTheDocument();
});

test("stands the logo in when a listing has no photos", () => {
  const { container } = renderDetail(makeListing({ images: [] }));

  expect(container.querySelector("img")).toBeNull();
  expect(container.querySelector('use[href="/icons.svg#brand-mark"]')).not.toBeNull();
});

test("says so when the listing cannot be loaded", () => {
  renderDetail(undefined, { isError: true, error: { status: 500, message: "Server exploded" } });
  expect(screen.getByText("Server exploded")).toBeInTheDocument();

  cleanup();
  renderDetail(undefined, { isPending: true });
  expect(document.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
});
