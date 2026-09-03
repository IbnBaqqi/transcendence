import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AccountActions } from "../components/objects/AccountActions";
import { useDeleteUser, useReinstateUser, useSuspendUser } from "../api/adminUsers";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import type { AdminUser, User } from "../api/types";

vi.mock("../api/adminUsers", () => ({
  useSuspendUser: vi.fn(),
  useReinstateUser: vi.fn(),
  useDeleteUser: vi.fn(),
}));

const TARGET_ID = "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6";
const ADMIN_ID = "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34";

const suspendMutate = vi.fn();
const reinstateMutate = vi.fn();
const deleteMutate = vi.fn();

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
  vi.mocked(useDeleteUser).mockReturnValue(
    stub({ mutate: deleteMutate }) as unknown as ReturnType<typeof useDeleteUser>,
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
  // Every button, not just the first: delete is the one that cannot be undone.
  expect(screen.queryAllByRole("button")).toHaveLength(0);
});

// Deletion is the one state with no way back.
test("offers nothing on a deleted account", () => {
  renderActions(makeAccount({ status: "deleted" }));
  expect(screen.queryAllByRole("button")).toHaveLength(0);
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

describe("deleting", () => {
  async function openDelete(account = makeAccount()) {
    renderActions(account);
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
  }

  // In a list of adjacent rows a yes/no dialog confirms the act, never the
  // target. Naming the account is what makes the admin look at which row it is.
  test("names the account it is about", async () => {
    await openDelete();
    expect(screen.getByLabelText("Type forager to confirm")).toBeInTheDocument();
  });

  test("says the account is anonymised rather than erased", async () => {
    await openDelete();
    expect(screen.getByText(/anonymised, not erased/)).toBeInTheDocument();
  });

  test("stays disabled until the username is typed", async () => {
    await openDelete();
    await userEvent.type(
      screen.getByRole("textbox", { name: "Reason (kept in the audit trail)" }),
      "spam",
    );
    expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
  });

  test("a case-only mismatch is not a match", async () => {
    await openDelete();
    await userEvent.type(screen.getByLabelText("Type forager to confirm"), "Forager");
    await userEvent.type(
      screen.getByRole("textbox", { name: "Reason (kept in the audit trail)" }),
      "spam",
    );
    expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
  });

  test("a matching name still needs a reason", async () => {
    await openDelete();
    await userEvent.type(screen.getByLabelText("Type forager to confirm"), "forager");
    expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
  });

  test("deletes once the name matches and a reason is given", async () => {
    await openDelete();
    await userEvent.type(screen.getByLabelText("Type forager to confirm"), "forager");
    await userEvent.type(
      screen.getByRole("textbox", { name: "Reason (kept in the audit trail)" }),
      "Repeatedly listing items they do not have",
    );
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));

    expect(deleteMutate).toHaveBeenCalledWith(
      {
        userId: TARGET_ID,
        username: "forager",
        reason: "Repeatedly listing items they do not have",
      },
      expect.anything(),
    );
  });

  // The point of trimming is that a copy-pasted name with surrounding space
  // still matches - and then the request must carry the value that matched,
  // not the raw one, or the server answers 400 for a name typed correctly.
  test("sends the name it compared, not the raw input", async () => {
    await openDelete();
    await userEvent.type(screen.getByLabelText("Type forager to confirm"), "  forager  ");
    await userEvent.type(
      screen.getByRole("textbox", { name: "Reason (kept in the audit trail)" }),
      "spam",
    );
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));

    expect(deleteMutate).toHaveBeenCalledWith(
      { userId: TARGET_ID, username: "forager", reason: "spam" },
      expect.anything(),
    );
  });

  // A suspended account can still be deleted - suspension is not a terminal
  // state, so both actions stay on offer.
  test("is offered for a suspended account too", () => {
    renderActions(makeAccount({ status: "suspended" }));
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reinstate" })).toBeInTheDocument();
  });
});
