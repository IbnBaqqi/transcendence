import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { RequireAdmin } from "../components/layout/RequireAdmin";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import type { User, UserRole } from "../api/types";

function makeUser(role: UserRole): User {
  return {
    id: "u1",
    username: "forager",
    email: "f@example.com",
    role,
    has_password: true,
    providers: [],
  };
}

function renderGuarded(auth: Partial<AuthContextValue>) {
  const value: AuthContextValue = {
    user: null,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
    ...auth,
  };

  render(
    <AuthContext.Provider value={value}>
      <MemoryRouter initialEntries={["/admin/listings"]}>
        <Routes>
          <Route element={<RequireAdmin />}>
            <Route path="/admin/listings" element={<p>the queue</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

test("lets an admin through", () => {
  renderGuarded({ user: makeUser("ADMIN") });
  expect(screen.getByText("the queue")).toBeInTheDocument();
});

test("shows a signed-in non-admin the 404", () => {
  renderGuarded({ user: makeUser("USER") });
  expect(screen.queryByText("the queue")).not.toBeInTheDocument();
  expect(screen.getByText("404 - Page not found")).toBeInTheDocument();
});

// A 404, not "you are not an admin" - the second one confirms the route exists.
test("shows a signed-out visitor the same 404, not a hint", () => {
  renderGuarded({ user: null });
  expect(screen.queryByText("the queue")).not.toBeInTheDocument();
  expect(screen.getByText("404 - Page not found")).toBeInTheDocument();
  expect(screen.queryByText(/admin/i)).not.toBeInTheDocument();
});

// The trap: a disabled//restoring session is neither signed in nor signed out.
// Falling through to the 404 here throws a reloading admin out of their page.
test("waits for the session rather than bouncing a reloading admin", () => {
  renderGuarded({ user: null, isLoading: true });
  expect(screen.queryByText("404 - Page not found")).not.toBeInTheDocument();
  expect(screen.getByText("Loading…")).toBeInTheDocument();
});
