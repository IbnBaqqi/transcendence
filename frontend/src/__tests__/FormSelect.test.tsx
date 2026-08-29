import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";

import { FormSelect } from "../components/forms/FormSelect";
import { useCategories } from "../api/categories";
import type { Category } from "../api/types";

vi.mock("../api/categories", async () => {
  const actual = await vi.importActual<typeof import("../api/categories")>("../api/categories");
  return { ...actual, useCategories: vi.fn() };
});

const TREE: Category[] = [
  { slug: "berries", name: "Berries", children: [] },
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

function Harness() {
  const form = useForm({ defaultValues: { category: "" } });
  return (
    <FormProvider {...form}>
      <FormSelect name="category" isEditing />
    </FormProvider>
  );
}

function renderWith(state: Partial<ReturnType<typeof useCategories>>) {
  vi.mocked(useCategories).mockReturnValue(state as ReturnType<typeof useCategories>);
  return render(<Harness />);
}

describe("FormSelect", () => {
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
