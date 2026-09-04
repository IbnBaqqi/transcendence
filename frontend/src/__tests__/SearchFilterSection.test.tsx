import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { SearchFilterSection } from "../components/forms/SearchFilterSection";
import { useCategories, categoryNames, useLocalizedCategoryNames } from "../api/categories";
import { emptyFilters } from "../lib/searchFilters";
import type { Category } from "../api/types";

vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn(), useLocalizedCategoryNames: vi.fn() };
});

const TREE: Category[] = [{ slug: "mushrooms", name: "Mushrooms", children: [] }];

function renderPanel(onApply = vi.fn(), onClear = vi.fn()) {
  vi.mocked(useCategories).mockReturnValue({
    data: TREE,
    isPending: false,
    isError: false,
  } as ReturnType<typeof useCategories>);
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(categoryNames(TREE));

  render(<SearchFilterSection initial={emptyFilters} onApply={onApply} onClear={onClear} />);
  return { onApply, onClear, user: userEvent.setup() };
}

describe("SearchFilterSection", () => {
  test("applies the filter set as strings, with an empty price box as no bound", async () => {
    const { onApply, user } = renderPanel();

    await user.type(screen.getByLabelText("Keyword"), "chanterelle");
    await user.selectOptions(screen.getByLabelText("Category"), "mushrooms");
    await user.type(screen.getByLabelText("Min price"), "5");
    await user.click(screen.getByRole("button", { name: "Apply filters" }));

    expect(onApply).toHaveBeenCalledWith({
      keyword: "chanterelle",
      category: "mushrooms",
      location: "",
      min_price: "5",
      max_price: "",
      sort: "newest",
    });
  });

  test("an over-long keyword blocks the submit", async () => {
    const { onApply, user } = renderPanel();

    // Set in one event rather than typed: user.type dispatches per character,
    // so 201 of them means 201 re-renders and 201 validations for a test about
    // the length rule. That cost is what made this file the first to time out
    // when the suite's workers compete for CPU.
    fireEvent.change(screen.getByLabelText("Keyword"), { target: { value: "a".repeat(201) } });
    await user.click(screen.getByRole("button", { name: "Apply filters" }));

    expect(await screen.findByText("Search text is too long")).toBeInTheDocument();
    expect(onApply).not.toHaveBeenCalled();
  });

  // Mirrors the backend rule, so the request that would 400 is never sent.
  test("an inverted price range blocks the submit", async () => {
    const { onApply, user } = renderPanel();

    await user.type(screen.getByLabelText("Min price"), "20");
    await user.type(screen.getByLabelText("Max price"), "5");
    await user.click(screen.getByRole("button", { name: "Apply filters" }));

    expect(
      await screen.findByText("Max price must not be lower than min price"),
    ).toBeInTheDocument();
    expect(onApply).not.toHaveBeenCalled();
  });
});
