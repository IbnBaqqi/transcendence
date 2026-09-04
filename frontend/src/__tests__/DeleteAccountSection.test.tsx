import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DeleteAccountSection } from "../components/forms/DeleteAccountSection";
import { useDeleteAccount } from "../api/profile";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import type { User } from "../api/types";

vi.mock("../api/profile", () => ({ useDeleteAccount: vi.fn() }));

const mutateAsync = vi.fn();
const logout = vi.fn();
const onClose = vi.fn();

function setMutation(over: Record<string, unknown> = {}) {
  vi.mocked(useDeleteAccount).mockReturnValue({
    mutateAsync,
    isPending: false,
    isError: false,
    error: null,
    ...over,
  } as unknown as ReturnType<typeof useDeleteAccount>);
}

const VIEWER: User = {
  id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
  username: "forager",
  email: "f@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

beforeEach(() => {
  mutateAsync.mockResolvedValue(undefined);
  setMutation();
});

function renderSection(user: User | null = VIEWER) {
  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout,
    restoreSession: vi.fn(),
  };
  render(
    <AuthContext.Provider value={auth}>
      <DeleteAccountSection onClose={onClose} />
    </AuthContext.Provider>,
  );
}

const confirmField = () => screen.getByLabelText("Type forager to confirm");
const deleteButton = () => screen.getByRole("button", { name: "Delete Account" });

// The guard the API imposes, and the only thing between a mis-click and an
// account that cannot be brought back.
test("stays disabled until the username is typed", () => {
  renderSection();
  expect(deleteButton()).toBeDisabled();
});

test("a case-only mismatch is not a match", async () => {
  renderSection();
  await userEvent.type(confirmField(), "Forager");
  await waitFor(() => expect(deleteButton()).toBeDisabled());
});

test("a near miss is not a match", async () => {
  renderSection();
  await userEvent.type(confirmField(), "forage");
  await waitFor(() => expect(deleteButton()).toBeDisabled());
});

test("sends the confirmation once it matches exactly", async () => {
  renderSection();
  await userEvent.type(confirmField(), "forager");
  await waitFor(() => expect(deleteButton()).toBeEnabled());
  await userEvent.click(deleteButton());

  await waitFor(() => expect(mutateAsync).toHaveBeenCalledWith("forager"));
});

// The account is gone, so the session is too - and logout is what clears the
// token, the user and every cached query scoped to it.
test("ends the session after the account is deleted", async () => {
  renderSection();
  await userEvent.type(confirmField(), "forager");
  await waitFor(() => expect(deleteButton()).toBeEnabled());
  await userEvent.click(deleteButton());

  await waitFor(() => expect(logout).toHaveBeenCalled());
});

// A failure must not close the modal and quietly leave the account standing.
test("surfaces a failure rather than swallowing it", () => {
  setMutation({
    isError: true,
    error: { status: 400, message: "Type your username exactly to confirm" },
  });
  renderSection();

  expect(screen.getByRole("alert")).toHaveTextContent("Type your username exactly to confirm");
});

// The modal is mounted at app root, so it outlives Profile's auth guard and the
// session can end underneath it - a background token refresh failing sets the
// user to null. The confirmation rule is value === username, so with no name to
// confirm against an empty box would satisfy the one control this form is for.
test("offers no form at all once the session has ended", () => {
  renderSection(null);

  expect(screen.queryByRole("button", { name: "Delete Account" })).not.toBeInTheDocument();
  expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  // Not a heading over an empty box: the state is rare, so it has to explain
  // itself rather than read as the modal having broken.
  expect(
    screen.getByText("You're signed out. Log in again to delete your account."),
  ).toBeInTheDocument();
});

// The same root cause with a visible symptom: the label interpolates the name.
test("never renders the label with an empty name", () => {
  renderSection(null);
  expect(screen.queryByText(/Type\s+to confirm/)).not.toBeInTheDocument();
});

// Deletion anonymises; it does not erase. The copy must not promise otherwise.
test("says the account is anonymised, not erased", () => {
  renderSection();
  expect(screen.getByText(/anonymised, not erased/)).toBeInTheDocument();
});
