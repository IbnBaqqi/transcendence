import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";

import AppRouter from "../routes";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { useOwnProfile } from "../api/profile";
import type { User, UserRole } from "../api/types";

vi.mock("../api/profile", () => ({ useOwnProfile: vi.fn() }));

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

beforeEach(() => {
  vi.mocked(useOwnProfile).mockReturnValue({ data: undefined } as ReturnType<typeof useOwnProfile>);
  window.history.pushState({}, "", "/admin/listings");
});

function renderApp(user: User | null) {
  const value: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={value}>
        <ModalProvider>
          <AppRouter />
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

// RequireAdmin is tested on its own; this asserts the route actually sits
// behind it. Without this, deleting the wrapper in routes/index.tsx breaks
// nothing and every other admin test still passes.
test("/admin/listings is behind the guard for an ordinary user", () => {
  renderApp(makeUser("USER"));
  expect(screen.queryByRole("heading", { name: "Moderation queue" })).not.toBeInTheDocument();
  expect(screen.getByText("404 - Page not found")).toBeInTheDocument();
});

test("/admin/listings is behind the guard for a visitor", () => {
  renderApp(null);
  expect(screen.queryByRole("heading", { name: "Moderation queue" })).not.toBeInTheDocument();
  expect(screen.getByText("404 - Page not found")).toBeInTheDocument();
});

test("an admin reaches the queue", () => {
  renderApp(makeUser("ADMIN"));
  expect(screen.getByRole("heading", { name: "Moderation queue" })).toBeInTheDocument();
});
