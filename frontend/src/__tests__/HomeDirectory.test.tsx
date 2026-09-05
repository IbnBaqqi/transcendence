import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { HomeDirectory } from "../components/objects/HomeDirectory";
import { useCategories, useLocalizedCategoryNames } from "../api/categories";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import type { Category } from "../api/types";

vi.mock("../api/categories");

const CATEGORIES: Category[] = [
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelle", name: "Chanterelle", children: [] }],
  },
  { slug: "berries", name: "Berries", children: [] },
];

beforeEach(() => {
  vi.mocked(useCategories).mockReturnValue({ data: CATEGORIES } as unknown as ReturnType<
    typeof useCategories
  >);
  vi.mocked(useLocalizedCategoryNames).mockReturnValue((slug: string) => `name:${slug}`);
});

function renderDirectory(user: AuthContextValue["user"]) {
  const value: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
  render(
    <AuthContext.Provider value={value}>
      <MemoryRouter>
        <HomeDirectory />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

const USER: AuthContextValue["user"] = {
  id: "u1",
  username: "forager",
  email: "f@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

// The point of the column: /search reads its filters from the URL, so a link
// is a whole category browse. If these stopped being real URLs the page would
// still render and the feature would be gone.
test("links each top-level category into a filtered search", () => {
  renderDirectory(null);

  expect(screen.getByRole("link", { name: "name:mushrooms" })).toHaveAttribute(
    "href",
    "/search?category=mushrooms",
  );
  expect(screen.getByRole("link", { name: "name:berries" })).toHaveAttribute(
    "href",
    "/search?category=berries",
  );
  // Children are not listed - the directory is a way in, not the whole tree.
  expect(screen.queryByRole("link", { name: "name:chanterelle" })).not.toBeInTheDocument();
});

test("keeps the account column for people who have one", () => {
  renderDirectory(null);
  expect(screen.queryByRole("link", { name: "Orders" })).not.toBeInTheDocument();

  cleanup();
  renderDirectory(USER);
  expect(screen.getByRole("link", { name: "Orders" })).toHaveAttribute("href", "/orders");
});
