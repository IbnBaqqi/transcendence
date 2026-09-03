import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AccountActions } from "../components/objects/AccountActions";
import { useReinstateUser, useSuspendUser } from "../api/adminUsers";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import type { AdminUser, User } from "../api/types";

vi.mock("../api/adminUsers", () => ({
  useSuspendUser: vi.fn(),
  useReinstateUser: vi.fn(),
}));

const TARGET_ID = "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6";
const ADMIN_ID = "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34";

const suspendMutate = vi.fn();
const reinstateMutate = vi.fn();

function stub(overrides: Record<string, unknown> = {}) {
  return {
    isPending: false,
    isError: false,
    error: null,
    ...overrides,
  };
}

function setMutations(overrides: Record<string, unknown> = {}) {
  vi.mocked(useSuspendUser).mockReturnValue(
    stub({ mutate: suspendMutate, ...overrides }) as unknown as ReturnType<typeof useSuspendUser>,
  );
  vi.mocked(useReinstateUser).mockReturnValue(
    stub({ mutate: reinstateMutate }) as unknown as ReturnType<typeof useReinstateUser>,
  );
}

beforeEach(() => setMutations());

function makeAccount(overrides: Partial<AdminUser> = {}): AdminUser {
  return {
    id: TARGET_ID,
    username: "forager",
    email: "f@example.com",
    role: "USER",
    status: "active",
    created_at: "2026-08-01T00:00:00Z",
    ...overrides,
  };
}

const VIEWER: User = {
  id: ADMIN_ID,
  username: "moderator",
  email: "m@example.com",
  role: "ADMIN",
  has_password: true,
  providers: [],
};

function renderActions(account: AdminUser, viewer: User | null = VIEWER) {
  const auth: AuthContextValue = {
    user: viewer,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
  render(
    <AuthContext.Provider value={auth}>
      <AccountActions account={account} />
    </AuthContext.Provider>,
  );
}

// Suspending yourself would lock you out of the endpoint that undoes it, so
// the API answers 403. Not offering the button is the point; reporting the
// error would be too late.
test("offers nothing on your own row", () => {
  renderActions(makeAccount({ id: ADMIN_ID }));
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});

// Deletion is the one state with no way back.
test("offers nothing on a deleted account", () => {
  renderActions(makeAccount({ status: "deleted" }));
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});

test("offers suspend for an active account, and not reinstate", () => {
  renderActions(makeAccount({ status: "active" }));
  expect(screen.getByRole("button", { name: "Suspend" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Reinstate" })).not.toBeInTheDocument();
});

test("offers reinstate for a suspended account, and not suspend", () => {
  renderActions(makeAccount({ status: "suspended" }));
  expect(screen.getByRole("button", { name: "Reinstate" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Suspend" })).not.toBeInTheDocument();
});

test("will not suspend without a reason", async () => {
  renderActions(makeAccount());
  await userEvent.click(screen.getByRole("button", { name: "Suspend" }));
  expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
});

test("will not accept whitespace as a reason", async () => {
  renderActions(makeAccount());
  await userEvent.click(screen.getByRole("button", { name: "Suspend" }));
  await userEvent.type(screen.getByRole("textbox"), "   ");
  expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
});

test("suspends with the trimmed reason", async () => {
  renderActions(makeAccount());
  await userEvent.click(screen.getByRole("button", { name: "Suspend" }));
  await userEvent.type(screen.getByRole("textbox"), "  Listing items they lack  ");
  await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

  expect(suspendMutate).toHaveBeenCalledWith(
    { userId: TARGET_ID, reason: "Listing items they lack" },
    expect.anything(),
  );
});

// "This was a mistake" needs no justification the way a punishment does.
test("reinstates with no note at all", async () => {
  renderActions(makeAccount({ status: "suspended" }));
  await userEvent.click(screen.getByRole("button", { name: "Reinstate" }));
  await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

  expect(reinstateMutate).toHaveBeenCalledWith({ userId: TARGET_ID }, expect.anything());
});

test("tells the suspended user's side who reads the reason", async () => {
  renderActions(makeAccount());
  await userEvent.click(screen.getByRole("button", { name: "Suspend" }));
  expect(screen.getByLabelText("Reason (shown to them)")).toBeInTheDocument();
});

test("caps the reason at the backend's own limit", async () => {
  renderActions(makeAccount());
  await userEvent.click(screen.getByRole("button", { name: "Suspend" }));
  expect(screen.getByRole("textbox")).toHaveAttribute("maxlength", "500");
});

// A 409 is either "already suspended" or "the last active admin" - only the
// server's message tells them apart, so it has to reach the screen.
test("surfaces the server's own message on a conflict", () => {
  setMutations({ isError: true, error: { status: 409, message: "That is the last active admin" } });
  renderActions(makeAccount());
  expect(screen.getByRole("alert")).toHaveTextContent("That is the last active admin");
});
