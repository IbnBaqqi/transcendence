import { render, screen } from "@testing-library/react";

import Home from "../pages/Home";
import { useListings } from "../api/listings";
import type { Listing } from "../api/types";

// Replace the whole module with auto-generated mock functions. Home imports
// useListings from here, so it gets the fake instead of the real hook.
// No network, no React Query, no QueryClientProvider needed.
vi.mock("../api/listings");

// The type useListings actually returns, so we don't have to hand-write it.
type ListingsQuery = ReturnType<typeof useListings>;

// Home only reads data/isPending/isError, so we pass a partial object and
// tell TypeScript to treat it as the full query result.
function mockListings(state: Partial<ListingsQuery>) {
  vi.mocked(useListings).mockReturnValue(state as ListingsQuery);
}

// One listing to render in the "has data" case.
const sample: Listing = {
  id: 1,
  title: "Golden Chanterelles",
  description: "Freshly foraged this morning.",
  category: "mushrooms",
  price: 18,
  quantity: 4,
  unit: "kg",
};

// A second, deliberately different listing: different id (so keys are unique),
// different title (so we can assert both rendered), different unit (so the
// "available" lines don't collide in getByText).
const secondSample: Listing = {
  id: 2,
  title: "Wild Blueberries",
  description: "Hand-picked from a sunny hillside.",
  category: "berries",
  price: 7.5,
  quantity: 10,
  unit: "litre",
};

describe("Home", () => {
  test("shows skeleton placeholders while the query is pending", () => {
    mockListings({ data: undefined, isPending: true, isError: false });
    const { container } = render(<Home />);
    expect(container.querySelectorAll(".skeleton")).toHaveLength(3);
  });

  test("shows an error message when the query fails", () => {
    mockListings({ data: undefined, isPending: false, isError: true });
    render(<Home />);
    expect(screen.getByText(/couldn't load listings/i)).toBeInTheDocument();
  });

  test("shows an empty message when there are no listings", () => {
    mockListings({ data: [], isPending: false, isError: false });
    render(<Home />);
    expect(screen.getByText("No listings yet.")).toBeInTheDocument();
  });

  test("renders a card for each listing", () => {
    mockListings({ data: [sample, secondSample], isPending: false, isError: false });
    render(<Home />);

    // both titles present -> we mapped the list, not just listings[0]
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
    expect(screen.getByText("Wild Blueberries")).toBeInTheDocument();

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
