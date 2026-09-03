import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import Home from "../pages/Home";
import { useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { makeListing } from "../test/factories";
import type { Listing, Paginated } from "../api/types";

// Replace the whole module with auto-generated mock functions. Home imports
// useSearchListings from here, so it gets the fake instead of the real hook.
// No network, no React Query, no QueryClientProvider needed.
vi.mock("../api/listings");
vi.mock("../api/categories");

// The type useSearchListings actually returns, so we don't have to hand-write it.
type ListingsQuery = ReturnType<typeof useSearchListings>;

// Home reads data.items now, so the fixtures carry a page rather than a bare
// array.
function page(items: Listing[]): Paginated<Listing> {
  return { items, total: items.length, page: 1, limit: 20, total_pages: 1 };
}

// Home only reads data/isPending/isError, so we pass a partial object and
// tell TypeScript to treat it as the full query result.
function mockListings(state: Partial<ListingsQuery>) {
  vi.mocked(useSearchListings).mockReturnValue(state as ListingsQuery);
}

const sample = makeListing();

// Deliberately different: id (so keys are unique), title (so we can assert
// both rendered), unit (so the "available" lines don't collide in getByText).
const secondSample = makeListing({
  id: "01a02305-b81d-764a-a738-d8c0642639de",
  title: "Wild Blueberries",
  description: "Hand-picked from a sunny hillside.",
  category: "berries",
  price: 7.5,
  quantity: 10,
  unit: "litre",
});

beforeEach(() => {
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(
    (slug: string) => ({ mushrooms: "Mushrooms", berries: "Berries" })[slug] ?? slug,
  );
});

// The cards are links now, so every render needs a router in scope.
function renderHome() {
  return render(
    <MemoryRouter>
      <Home />
    </MemoryRouter>,
  );
}

describe("Home", () => {
  test("shows skeleton placeholders while the query is pending", () => {
    mockListings({ data: undefined, isPending: true, isError: false });
    const { container } = renderHome();
    expect(container.querySelectorAll(".skeleton")).toHaveLength(3);
  });

  test("shows an error message when the query fails", () => {
    mockListings({ data: undefined, isPending: false, isError: true });
    renderHome();
    expect(screen.getByText(/couldn't load listings/i)).toBeInTheDocument();
  });

  test("shows an empty message when there are no listings", () => {
    mockListings({ data: page([]), isPending: false, isError: false });
    renderHome();
    expect(screen.getByText("No listings yet.")).toBeInTheDocument();
  });

  test("offers a way to the rest when one page is not the whole set", async () => {
    mockListings({
      data: { items: [sample], total: 47, page: 1, limit: 20, total_pages: 3 },
      isPending: false,
      isError: false,
    });
    renderHome();

    expect(screen.getByText("47 results")).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: "Next" }));

    expect(useSearchListings).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }));
  });

  test("renders a card for each listing", () => {
    mockListings({ data: page([sample, secondSample]), isPending: false, isError: false });
    renderHome();

    // both titles present -> we mapped the list, not just listings[0]
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
    expect(screen.getByText("Wild Blueberries")).toBeInTheDocument();

    expect(screen.getByText("Mushrooms")).toBeInTheDocument();
    expect(screen.getByText("Berries")).toBeInTheDocument();
    expect(screen.queryByText("mushrooms")).not.toBeInTheDocument();
    expect(screen.queryByText("berries")).not.toBeInTheDocument();

    // the card formats price + unit together, so match loosely
    expect(screen.getByText(/€18\.00/)).toBeInTheDocument();
    expect(screen.getByText(/€7\.50/)).toBeInTheDocument();

    // exactly two cards: catches dropped or duplicated items. <article> maps
    // to the "article" ARIA role, so this counts cards without a test id.
    expect(screen.getAllByRole("article")).toHaveLength(2);

    // and the empty-state message must NOT be showing
    expect(screen.queryByText("No listings yet.")).not.toBeInTheDocument();
  });
});
