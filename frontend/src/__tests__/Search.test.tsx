import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import Search from "../pages/Search";
import { useSearchListings } from "../api/listings";
import { useCategories, categoryNames, useLocalizedCategoryNames } from "../api/categories";
import { makeListing } from "../test/factories";
import type { Category, Listing, Paginated } from "../api/types";

vi.mock("../api/listings");
vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn(), useLocalizedCategoryNames: vi.fn() };
});

type ListingsQuery = ReturnType<typeof useSearchListings>;

const TREE: Category[] = [{ slug: "mushrooms", name: "Mushrooms", children: [] }];

function page(items: Listing[]): Paginated<Listing> {
  return { items, total: items.length, page: 1, limit: 20, total_pages: 1 };
}

// Shows what the address bar holds, so a click can be asserted on the URL.
function QueryProbe() {
  return <output data-testid="query">{useLocation().search}</output>;
}

function renderSearch(
  query: string,
  state: Partial<ListingsQuery> = { data: page([makeListing()]) },
) {
  vi.mocked(useSearchListings).mockReturnValue(state as ListingsQuery);
  vi.mocked(useCategories).mockReturnValue({
    data: TREE,
    isPending: false,
    isError: false,
  } as ReturnType<typeof useCategories>);
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(categoryNames(TREE));

  render(
    <MemoryRouter initialEntries={[`/search${query}`]}>
      <Routes>
        <Route
          path="/search"
          element={
            <>
              <Search />
              <QueryProbe />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
  return userEvent.setup();
}

describe("Search", () => {
  test("asks the API for what the URL says", () => {
    renderSearch("?category=mushrooms&min_price=5&page=2");

    expect(useSearchListings).toHaveBeenCalledWith(
      expect.objectContaining({ category: "mushrooms", min_price: 5, page: 2, limit: 20 }),
    );
  });

  test("dismissing a chip drops that filter and returns to page 1", async () => {
    const user = renderSearch("?keyword=bolete&category=mushrooms&page=3");

    await user.click(screen.getByRole("button", { name: "Remove filter: Mushrooms" }));

    // Exact, not toHaveTextContent: that matches substrings, so a leftover
    // &page=3 would pass.
    expect(screen.getByTestId("query").textContent).toBe("?keyword=bolete");
  });

  test("no results is a normal answer, not an error", () => {
    renderSearch("?keyword=nothing", { data: page([]) });

    expect(screen.getByText("No listings match these filters.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /try again/i })).not.toBeInTheDocument();
  });
});
