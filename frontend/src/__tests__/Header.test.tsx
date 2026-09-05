import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
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

// { user: null } rather than no argument: a bare renderHeader() uses the real
// AuthProvider, which starts mid-restore - so this asserted the restoring state
// while claiming to test the signed-out one.
test("offers login when signed out", () => {
  renderHeader({ user: null });
  expect(screen.getByText("?")).toBeInTheDocument();
});

test("shows who is signed in", () => {
  renderHeader({
    user: {
      id: "u1",
      username: "forager",
      email: "f@example.com",
      role: "USER",
      has_password: true,
      providers: [],
    },
  });
  // Single initial from the username - names live on the profile.
  expect(screen.getByText("F")).toBeInTheDocument();
});

// The bell is a notifications panel now, not a link, so it has nothing to show
// a visitor with no account - and asking on their behalf is a 401 every poll.
test("hangs the notification bell on a session, not on every visitor", () => {
  renderHeader({ user: null });
  expect(screen.queryByLabelText("Notifications")).not.toBeInTheDocument();

  cleanup();
  renderHeader({
    user: {
      id: "u1",
      username: "forager",
      email: "f@example.com",
      role: "USER",
      has_password: true,
      providers: [],
    },
  });
  expect(screen.getByLabelText("Notifications")).toBeInTheDocument();
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

const SIGNED_IN: Partial<AuthContextValue> = {
  user: {
    id: "u1",
    username: "or99",
    email: "o@example.com",
    role: "USER",
    has_password: true,
    providers: [],
  },
};

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

// The header is on every page, so it cannot early-return like a section can -
// only the slot that names the viewer holds. The brand assertion is what makes
// that difference the subject of the test rather than a side effect.
test("holds the identity slot but keeps the header while the session restores", () => {
  renderHeader({ isLoading: true });
  expect(screen.queryByLabelText("Menu")).not.toBeInTheDocument();
  expect(screen.getByText("Metsätori")).toBeInTheDocument();
});

// Each of these is the only navigation into its screen anywhere in app code,
// and every one of their accessible names comes from an aria-label alone. A
// rewrite that drops one leaves a section reachable only by typing the URL,
// with the suite still green - which is how /admin/orders was lost, and it
// took comparing ancestry by hand to notice. The signed-in links moved into
// UserMenu and are tested there for the same reason.
// The mark is the only thing in the top-left below sm, where the wordmark is
// hidden. It renders through the sprite because an external <use> with no
// fragment draws nothing in Firefox - which is how it was found. Named for the
// brand rather than "Home": the nav already has a Home link, and its only child
// is aria-hidden, so without the label this link has no accessible name.
test("gives the brand mark its own name, distinct from the Home link", () => {
  renderHeader({ user: null });

  const mark = screen.getByRole("link", { name: "Metsätori" });
  expect(mark).toHaveAttribute("href", "/");
  expect(mark.querySelector("use")).toHaveAttribute("href", "/icons.svg#brand-mark");
  expect(screen.getByRole("link", { name: "Home" })).not.toBe(mark);
});

test.each([
  ["Home", "/"],
  ["Search", "/search"],
  ["Add listing", "/addlisting"],
])("points %s at %s for every visitor", (name, href) => {
  renderHeader({ user: null });
  expect(screen.getByRole("link", { name })).toHaveAttribute("href", href);
});
