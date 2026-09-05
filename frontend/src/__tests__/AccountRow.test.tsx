import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AccountRow } from "../components/objects/AccountRow";
import { useUserHistory } from "../api/adminUsers";
import type { AdminUser, UserAction } from "../api/types";

vi.mock("../api/adminUsers", () => ({ useUserHistory: vi.fn() }));
// This file is about the row and its history; the actions have their own tests.
vi.mock("../components/objects/AccountActions", () => ({ AccountActions: () => null }));

const TARGET_ID = "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6";

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

function makeAction(overrides: Partial<UserAction> = {}): UserAction {
  return {
    id: "a1",
    action: "suspended",
    note: "Listing items they lack",
    moderator_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    created_at: "2026-08-02T00:00:00Z",
    ...overrides,
  };
}

function setHistory(history: UserAction[] | undefined, isError = false) {
  vi.mocked(useUserHistory).mockReturnValue({ data: history, isError } as ReturnType<
    typeof useUserHistory
  >);
}

function renderRow(account = makeAccount()) {
  render(
    <ul>
      <AccountRow account={account} />
    </ul>,
  );
}

beforeEach(() => setHistory([makeAction()]));

// A collapsed row passes "" so the query stays disabled - that is what makes a
// long list cost nothing beyond the list itself.
test("asks for nothing until the row is opened", () => {
  renderRow();
  expect(useUserHistory).toHaveBeenCalledWith("");
});

test("asks for that account once opened", async () => {
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(useUserHistory).toHaveBeenCalledWith(TARGET_ID);
});

test("keeps the history shut until asked", () => {
  renderRow();
  expect(screen.queryByText("Listing items they lack")).not.toBeInTheDocument();
});

test("shows what was done and why", async () => {
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getByText("Suspended")).toBeInTheDocument();
  expect(screen.getByText("Listing items they lack")).toBeInTheDocument();
});

// Nothing here is edited or deleted, so an account suspended and reinstated
// twice shows all four rows.
test("keeps every entry rather than the latest", async () => {
  setHistory([
    makeAction({ id: "a1", action: "reinstated", note: "" }),
    makeAction({ id: "a2", action: "suspended", note: "Second strike" }),
    makeAction({ id: "a3", action: "reinstated", note: "" }),
    makeAction({ id: "a4", action: "suspended", note: "First strike" }),
  ]);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getAllByText("Suspended")).toHaveLength(2);
  expect(screen.getAllByText("Reinstated")).toHaveLength(2);
});

// The audit row outlives its author deliberately.
test("says so when the moderator has deleted their account", async () => {
  setHistory([makeAction({ moderator_id: null })]);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getByText("Moderator has deleted their account")).toBeInTheDocument();
});

test("says when an account was never actioned", async () => {
  setHistory([]);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getByText("Never actioned.")).toBeInTheDocument();
});

// A query that has exhausted its retries still has data === undefined, so
// "loading" is indistinguishable from "gave up" unless isError is read.
test("says the history failed rather than loading forever", async () => {
  setHistory(undefined, true);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getByText("Couldn't load the history.")).toBeInTheDocument();
  expect(screen.queryByText("Loading…")).not.toBeInTheDocument();
});

// Keeping the history is the point of anonymising the row rather than dropping
// it, so a deleted account still has one to read - even though it has no
// actions left to offer.
test("offers the history of a deleted account", async () => {
  renderRow(makeAccount({ status: "deleted", username: "deleted-9c4e1b7a" }));
  await userEvent.click(screen.getByRole("button", { name: "Show history" }));
  expect(screen.getByText("Suspended")).toBeInTheDocument();
});

// The role endpoint writes `promoted` and `demoted` audit rows. Without a
// label for each, this panel renders the raw key - "adminUsers.action.promoted"
// - at whoever opens the history. Every UserActionKind needs a string in all
// three locales, and the union is what makes a missing one a build error.
test.each([
  ["promoted", "Made admin"],
  ["demoted", "Admin removed"],
])("names a %s row rather than showing its key", async (action, label) => {
  setHistory([makeAction({ action: action as UserAction["action"] })]);
  renderRow();

  await userEvent.click(screen.getByRole("button", { name: "Show history" }));

  expect(screen.getByText(label)).toBeInTheDocument();
  expect(screen.queryByText(/adminUsers\.action\./)).not.toBeInTheDocument();
});
