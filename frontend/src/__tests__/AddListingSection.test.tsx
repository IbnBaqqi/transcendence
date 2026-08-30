import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AddListingSection } from "../components/forms/AddListingSection";
import { useCategories } from "../api/categories";
import type { Category } from "../api/types";

vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn() };
});

const TREE: Category[] = [
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

function mockCategories(data: Category[] | undefined) {
  vi.mocked(useCategories).mockReturnValue({
    data,
    isPending: data === undefined,
    isError: false,
  } as ReturnType<typeof useCategories>);
}

describe("AddListingSection", () => {
  test("accepts a category that only became valid after the list arrived", async () => {
    mockCategories(undefined);
    const { rerender } = render(<AddListingSection />);

    expect(screen.getByRole("combobox")).toBeDisabled();

    mockCategories(TREE);
    rerender(<AddListingSection />);

    const select = screen.getByRole("combobox");
    await waitFor(() => expect(select).toBeEnabled());

    await userEvent.selectOptions(select, "chanterelles");
    await userEvent.tab();

    await waitFor(() => {
      expect(screen.queryByText("Choose a category from the list")).not.toBeInTheDocument();
    });
    expect(select).toHaveValue("chanterelles");
  });
});
