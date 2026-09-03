import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import Header from "../components/layout/Header";
import { useOwnProfile } from "../api/profile";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthProvider } from "../providers/AuthProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";

vi.mock("../api/profile", () => ({ useOwnProfile: vi.fn() }));

beforeEach(() => {
  vi.mocked(useOwnProfile).mockReturnValue({ data: undefined } as ReturnType<typeof useOwnProfile>);
});

function renderHeader(auth?: Partial<AuthContextValue>) {
  const value: AuthContextValue = {
    user: null,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
    ...auth,
  };
  // The client is outermost now: the header reads /me/profile for the avatar,
  // and AuthProvider clears the cache on logout, so both branches need one.
  render(
    <QueryClientProvider client={new QueryClient()}>
      {/* The real provider fires a session restore on mount; tests pass their
          own values unless they specifically want that round trip. */}
      {auth ? (
        <AuthContext.Provider value={value}>
          <ModalProvider>
            <MemoryRouter>
              <Header />
            </MemoryRouter>
          </ModalProvider>
        </AuthContext.Provider>
      ) : (
        <AuthProvider>
          <ModalProvider>
            <MemoryRouter>
              <Header />
            </MemoryRouter>
          </ModalProvider>
        </AuthProvider>
      )}
    </QueryClientProvider>,
  );
}

test("renders the brand name", () => {
  renderHeader();
  expect(screen.getByText("Metsätori")).toBeInTheDocument();
});

test("offers login when signed out", () => {
  renderHeader();
  expect(screen.getByText("?")).toBeInTheDocument();
});

test("shows who is signed in", () => {
  renderHeader({
    user: { id: "u1", username: "forager", email: "f@example.com", role: "USER" },
  });
  // Single initial from the username - names live on the profile.
  expect(screen.getByText("F")).toBeInTheDocument();
});

test("places the language switcher at the far left of the nav", () => {
  renderHeader();
  const nav = screen.getByRole("navigation");
  const switcher = screen.getByRole("button", { name: "Language" });
  const home = screen.getByRole("link", { name: "Home" });
  expect(nav.firstElementChild?.contains(switcher)).toBe(true);
  // ...and still left of the Home link
  expect(switcher.compareDocumentPosition(home) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});

const SIGNED_IN = { user: { id: "u1", username: "or99", email: "o@example.com", role: "user" } };

test("wears the signed-in user's picture when they have one", () => {
  vi.mocked(useOwnProfile).mockReturnValue({
    data: { avatar_url: "/uploads/me.png" },
  } as ReturnType<typeof useOwnProfile>);

  renderHeader(SIGNED_IN);

  expect(document.querySelector("header img")).toHaveAttribute("src", "/uploads/me.png");
});

// Initials are the default avatar, so the corner must not sit empty for
// somebody who has never uploaded anything.
test("falls back to initials when no picture is set", () => {
  renderHeader(SIGNED_IN);

  expect(document.querySelector("header img")).toBeNull();
  expect(screen.getByText("O")).toBeInTheDocument();
});

// A signed-out visitor has no profile to fetch, and asking anyway costs a 401
// plus a refresh attempt on the interceptor.
test("does not ask for a profile while signed out", () => {
  renderHeader({ user: null });

  expect(useOwnProfile).toHaveBeenLastCalledWith({ enabled: false });
});
