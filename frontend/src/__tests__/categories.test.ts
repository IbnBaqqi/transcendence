import { categoryNames, flattenCategories } from "../api/categories";
import type { Category } from "../api/types";

const TREE: Category[] = [
  { slug: "berries", name: "Berries", children: [] },
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

describe("flattenCategories", () => {
  test("keeps parents and children in one list, marked by depth", () => {
    expect(flattenCategories(TREE)).toEqual([
      { slug: "berries", name: "Berries", depth: 0 },
      { slug: "mushrooms", name: "Mushrooms", depth: 0 },
      { slug: "chanterelles", name: "Chanterelles", depth: 1 },
    ]);
  });

  test("a child follows its own parent, not the end of the list", () => {
    const flat = flattenCategories(TREE);
    expect(flat.findIndex((c) => c.slug === "chanterelles")).toBe(
      flat.findIndex((c) => c.slug === "mushrooms") + 1,
    );
  });

  test("no categories is an empty list, not a crash", () => {
    expect(flattenCategories([])).toEqual([]);
  });
});

describe("categoryNames", () => {
  test("maps a slug to its display name, children included", () => {
    const nameOf = categoryNames(TREE);
    expect(nameOf("mushrooms")).toBe("Mushrooms");
    expect(nameOf("chanterelles")).toBe("Chanterelles");
  });

  test("falls back to the slug before the categories arrive", () => {
    expect(categoryNames([])("mushrooms")).toBe("mushrooms");
  });

  test("falls back to the slug for a category that no longer exists", () => {
    expect(categoryNames(TREE)("truffles")).toBe("truffles");
  });
});
