import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";

import { CategorySelect } from "../components/forms/CategorySelect";
import { useCategories, categoryNames, useLocalizedCategoryNames } from "../api/categories";
import type { Category } from "../api/types";

vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn(), useLocalizedCategoryNames: vi.fn() };
});

const TREE: Category[] = [
  { slug: "berries", name: "Berries", children: [] },
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

function Harness({ onSubmit }: { onSubmit?: (values: { category: string }) => void }) {
  const form = useForm({ defaultValues: { category: "" } });
  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit((values) => onSubmit?.(values))}>
        <CategorySelect name="category" ariaLabel="Category" isEditing />
        <button type="submit">Save</button>
      </form>
    </FormProvider>
  );
}

function renderWith(
  state: Partial<ReturnType<typeof useCategories>>,
  onSubmit?: (values: { category: string }) => void,
) {
  vi.mocked(useCategories).mockReturnValue(state as ReturnType<typeof useCategories>);
  // CategorySelect resolves display labels through the localized hook; derive
  // them from whatever data this render is pretending to have.
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(categoryNames(state.data ?? []));
  return render(<Harness onSubmit={onSubmit} />);
}

describe("CategorySelect", () => {
  test("lists every category, with children under their parent", () => {
    renderWith({ data: TREE, isPending: false, isError: false });

    const options = screen.getAllByRole("option");
    expect(options.map((o) => o.textContent?.trim())).toEqual([
      "Choose a category",
      "Berries",
      "Mushrooms",
      "Chanterelles",
    ]);
  });

  test("the value submitted is the slug, not the name", () => {
    renderWith({ data: TREE, isPending: false, isError: false });

    expect(screen.getByRole("option", { name: /Mushrooms/ })).toHaveValue("mushrooms");
    expect(screen.getByRole("option", { name: /Chanterelles/ })).toHaveValue("chanterelles");
  });

  test("nothing is preselected, so the field stays required", () => {
    renderWith({ data: TREE, isPending: false, isError: false });

    expect(screen.getByRole("combobox")).toHaveValue("");
  });

  test("choosing an option puts its slug into the form", async () => {
    const onSubmit = vi.fn();
    renderWith({ data: TREE, isPending: false, isError: false }, onSubmit);

    await userEvent.selectOptions(screen.getByRole("combobox"), "chanterelles");
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).toHaveBeenCalledWith({ category: "chanterelles" });
  });

  test("an empty but successful response disables the field rather than offering nothing", () => {
    renderWith({ data: [], isPending: false, isError: false });

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByText(/No categories are available/)).toBeInTheDocument();
    expect(screen.queryByText(/Could not load/)).not.toBeInTheDocument();
  });

  test("a failed request offers a retry rather than telling the user to reload", async () => {
    const refetch = vi.fn();
    vi.mocked(useCategories).mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
      refetch,
    } as unknown as ReturnType<typeof useCategories>);
    render(<Harness />);

    await userEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(refetch).toHaveBeenCalled();
  });

  test("the control has an accessible name", () => {
    renderWith({ data: TREE, isPending: false, isError: false });

    expect(screen.getByRole("combobox", { name: "Category" })).toBeInTheDocument();
  });

  test("the unavailable message is announced and tied to the control", () => {
    renderWith({ data: [], isPending: false, isError: false });

    const message = screen.getByRole("alert");
    expect(message).toHaveTextContent(/No categories are available/);
    expect(screen.getByRole("combobox")).toHaveAttribute("aria-describedby", message.id);
  });

  test("a failed request disables the field and says so", () => {
    renderWith({ data: undefined, isPending: false, isError: true });

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByText(/Could not load categories/)).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /Berries/ })).not.toBeInTheDocument();
  });

  test("a pending request disables the field so nothing is submitted early", () => {
    renderWith({ data: undefined, isPending: true, isError: false });

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByRole("option", { name: /Loading/ })).toBeInTheDocument();
  });
});
