import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import AdminUsers from "../pages/AdminUsers";
import { useAdminUsers } from "../api/adminUsers";
import type { AdminUser, PaginatedAdminUsers } from "../api/types";

vi.mock("../api/adminUsers", () => ({ useAdminUsers: vi.fn() }));

function makeUser(overrides: Partial<AdminUser> = {}): AdminUser {
  return {
    id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
    username: "forager",
    email: "f@example.com",
    role: "USER",
    status: "active",
    created_at: "2026-08-01T00:00:00Z",
    ...overrides,
  };
}

function page(items: AdminUser[], total = items.length): PaginatedAdminUsers {
  return { items, total, page: 1, limit: 20, total_pages: 1 };
}

// Rendered rather than assigned to a module variable: writing during render is
// a side effect, and eslint is right to refuse it.
function ShowUrl() {
  return <span data-testid="url">{useLocation().search}</span>;
}

const url = () => screen.getByTestId("url").textContent;

function renderPage(query: Partial<ReturnType<typeof useAdminUsers>>, url = "/admin/users") {
  vi.mocked(useAdminUsers).mockReturnValue(query as ReturnType<typeof useAdminUsers>);
  render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route
          path="/admin/users"
          element={
            <>
              <AdminUsers />
              <ShowUrl />
            </>
          }
        />
      </Routes>
      ,
    </MemoryRouter>,
  );
}

// The literal types matter: widened to `boolean` these stop matching any one
// member of the UseQueryResult union and tsc rejects the object - while vitest
// runs it happily, which is why this is a typecheck problem, not a test one.
const loaded = (items: AdminUser[], total = items.length) => ({
  data: page(items, total),
  isPending: false as const,
  isError: false as const,
});

test("lists the accounts it is given", () => {
  renderPage(loaded([makeUser()]));
  expect(screen.getByText("forager")).toBeInTheDocument();
  expect(screen.getByText("f@example.com")).toBeInTheDocument();
});

// deleted-<id> is anonymisation, not a name: rendering it would put an id on
// screen where a person used to be.
test("a deleted account is named as deleted, not by its placeholder", () => {
  renderPage(loaded([makeUser({ status: "deleted", username: "deleted-9c4e1b7a" })]));
  expect(screen.getByText("Deleted user")).toBeInTheDocument();
  expect(screen.queryByText("deleted-9c4e1b7a")).not.toBeInTheDocument();
});

// The reason survives deletion on purpose - why an account was actioned is
// context an admin still wants.
test("shows why a suspended account was suspended", () => {
  renderPage(
    loaded([makeUser({ status: "suspended", suspension_reason: "Listing items they lack" })]),
  );
  expect(screen.getByText("Suspended: Listing items they lack")).toBeInTheDocument();
});

test("says when an account has never been seen", () => {
  renderPage(loaded([makeUser()]));
  expect(screen.getByText("Never seen")).toBeInTheDocument();
});

// The filters are the URL, so this is what makes a filtered view linkable.
test("choosing a filter writes it to the URL", async () => {
  renderPage(loaded([makeUser()]));
  await userEvent.selectOptions(screen.getByLabelText("Status"), "suspended");
  expect(url()).toBe("?status=suspended");
});

test("the selects show what the URL already says", () => {
  renderPage(loaded([makeUser()]), "/admin/users?role=ADMIN&status=deleted");
  expect(screen.getByLabelText("Role")).toHaveValue("ADMIN");
  expect(screen.getByLabelText("Status")).toHaveValue("deleted");
});

// Changing a filter while deep in the pages would otherwise strand the reader
// on a page that is valid and empty.
test("changing a filter drops the page number", async () => {
  renderPage(loaded([makeUser()]), "/admin/users?page=5");
  await userEvent.selectOptions(screen.getByLabelText("Role"), "ADMIN");
  expect(url()).toBe("?role=ADMIN");
});

test("an unknown filter in the URL is ignored rather than sent on", () => {
  renderPage(loaded([makeUser()]), "/admin/users?status=banana");
  expect(screen.getByLabelText("Status")).toHaveValue("");
});

// total, not the length of this page: an empty page past the end is not the
// same as nothing matching.
test("says so when nothing matches", () => {
  renderPage(loaded([], 0));
  expect(screen.getByText("No accounts match those filters.")).toBeInTheDocument();
});

test("offers a retry when the list will not load", async () => {
  const refetch = vi.fn();
  renderPage({ data: undefined, isPending: false, isError: true, error: new Error("x"), refetch });
  expect(screen.getByText("Couldn't load the accounts.")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Try again" }));
  expect(refetch).toHaveBeenCalled();
});
