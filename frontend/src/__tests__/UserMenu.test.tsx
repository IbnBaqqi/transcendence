import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { UserMenu } from "../components/layout/UserMenu";
import { useOwnProfile } from "../api/profile";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";

vi.mock("../api/profile", () => ({ useOwnProfile: vi.fn() }));

beforeEach(() => {
  vi.mocked(useOwnProfile).mockReturnValue({ data: undefined } as ReturnType<typeof useOwnProfile>);
});

const USER: AuthContextValue["user"] = {
  id: "u1",
  username: "forager",
  email: "f@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

function renderMenu(auth: Partial<AuthContextValue>) {
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
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={value}>
        <ModalProvider>
          <MemoryRouter>
            <UserMenu />
          </MemoryRouter>
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

const openMenu = (user: ReturnType<typeof userEvent.setup>) =>
  user.click(screen.getByRole("button", { name: "Menu" }));

test("keeps the panel closed until the trigger is pressed, and says so", async () => {
  const user = userEvent.setup();
  renderMenu({ user: USER });

  const trigger = screen.getByRole("button", { name: "Menu" });
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(screen.queryByRole("group")).not.toBeInTheDocument();

  await user.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  expect(screen.getByRole("group", { name: "Menu" })).toBeInTheDocument();
});

// Each of these is the only navigation into its screen anywhere in app code,
// and every accessible name comes from the link text alone. A rewrite that
// drops one leaves a section reachable only by typing the URL, with the suite
// still green - which is how /admin/orders was lost once already.
test.each([
  ["Profile", "/profile"],
  ["Orders", "/orders"],
  ["Following", "/following"],
])("points %s at %s once signed in, and hides it from a visitor", async (name, href) => {
  const user = userEvent.setup();
  renderMenu({ user: USER });

  await openMenu(user);
  expect(screen.getByRole("link", { name })).toHaveAttribute("href", href);

  cleanup();
  renderMenu({ user: null });
  await openMenu(user);
  expect(screen.queryByRole("link", { name })).not.toBeInTheDocument();
});

// The links are UX, not a lock - /admin/* is guarded by RequireAdmin and every
// endpoint under it by RequireRole(ADMIN). This just stops the menu
// advertising a section three quarters of users cannot open.
test("offers the admin section to an admin and to nobody else", async () => {
  const user = userEvent.setup();
  renderMenu({ user: { ...USER, role: "ADMIN" } });

  await openMenu(user);
  expect(screen.getByRole("link", { name: "Admin" })).toHaveAttribute("href", "/admin/listings");
  expect(screen.getByRole("link", { name: "Accounts" })).toHaveAttribute("href", "/admin/users");
  expect(screen.getByRole("link", { name: "Order admin" })).toHaveAttribute(
    "href",
    "/admin/orders",
  );

  cleanup();
  renderMenu({ user: USER });
  await openMenu(user);
  expect(screen.queryByRole("link", { name: "Admin" })).not.toBeInTheDocument();
});

test("offers a visitor the two ways in", async () => {
  const user = userEvent.setup();
  renderMenu({ user: null });

  await openMenu(user);
  expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Register" })).toBeInTheDocument();
});

// The panel unmounts under the keyboard, so without the handoff focus falls
// back to the top of the document.
test("closes on Escape and hands focus back to the trigger", async () => {
  const user = userEvent.setup();
  renderMenu({ user: USER });

  await openMenu(user);
  await user.keyboard("{Escape}");

  expect(screen.queryByRole("group")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Menu" })).toHaveFocus();
});

// A menu offering "Log In" is a claim about the viewer, and it is the wrong
// one on every reload by a signed-in user.
test("shows no trigger at all while the session is still restoring", () => {
  renderMenu({ isLoading: true });

  expect(screen.queryByRole("button", { name: "Menu" })).not.toBeInTheDocument();
});
