import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import Search from "../pages/Search";
import { useSearchListings } from "../api/listings";
import { useCategories, useLocalizedCategoryNames } from "../api/categories";
import type { Category } from "../api/types";

// Only the network hooks are faked - toSearchParams and toQueryString stay
// real, so the URL <-> params translation under test is the real thing.
vi.mock("../api/listings", async () => {
  const actual = await vi.importActual<typeof import("../api/listings")>("../api/listings");
  return { ...actual, useSearchListings: vi.fn() };
});
vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn(), useLocalizedCategoryNames: vi.fn() };
});

type ListingsQuery = ReturnType<typeof useSearchListings>;

const TREE: Category[] = [
  { slug: "berries", name: "Berries", children: [] },
  { slug: "mushrooms", name: "Mushrooms", children: [] },
];

function renderSearch(initialUrl: string) {
  vi.mocked(useCategories).mockReturnValue({ data: TREE } as ReturnType<typeof useCategories>);
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(
    (slug: string) => TREE.find((c) => c.slug === slug)?.name ?? slug,
  );
  vi.mocked(useSearchListings).mockReturnValue({
    data: { items: [], total: 0, page: 1, limit: 20, total_pages: 1 },
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  } as unknown as ListingsQuery);

  return render(
    <MemoryRouter initialEntries={[initialUrl]}>
      <Search />
    </MemoryRouter>,
  );
}

function lastParams() {
  return vi.mocked(useSearchListings).mock.calls.at(-1)?.[0];
}

describe("Search", () => {
  test("dismissing a chip removes only that filter from the URL", async () => {
    const user = userEvent.setup();
    renderSearch("/search?category=berries&location=Helsinki");

    await user.click(screen.getByRole("button", { name: /remove helsinki filter/i }));

    expect(lastParams()).toEqual(expect.objectContaining({ category: "berries" }));
    expect(lastParams()?.location).toBeUndefined();
  });

  test("changing a filter resets the page to 1", async () => {
    const user = userEvent.setup();
    renderSearch("/search?category=berries&page=3");

    await user.selectOptions(screen.getByLabelText("Category"), "mushrooms");

    expect(lastParams()).toEqual(expect.objectContaining({ category: "mushrooms", page: 1 }));
  });
});
