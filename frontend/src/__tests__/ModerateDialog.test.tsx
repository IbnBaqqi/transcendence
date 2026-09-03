import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ModerateDialog } from "../components/objects/ModerateDialog";
import { useModerateListing } from "../api/moderation";

vi.mock("../api/moderation", () => ({ useModerateListing: vi.fn() }));

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

const mutate = vi.fn();

function setMutation(overrides: Record<string, unknown> = {}) {
  vi.mocked(useModerateListing).mockReturnValue({
    mutate,
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
    data: undefined,
    ...overrides,
  } as unknown as ReturnType<typeof useModerateListing>);
}

beforeEach(() => setMutation());

function renderDialog(removed = false) {
  render(<ModerateDialog listingId={LISTING_ID} removed={removed} />);
}

// removed_at decides what is offered. Showing all three regardless is how you
// generate 409s on purpose - the API refuses remove on an already removed
// listing and restore on one that is not.
test("offers remove and dismiss for a listing that is still up", () => {
  renderDialog(false);
  expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Restore" })).not.toBeInTheDocument();
});

test("offers only restore for a listing already removed", () => {
  renderDialog(true);
  expect(screen.getByRole("button", { name: "Restore" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Dismiss" })).not.toBeInTheDocument();
});

// The API requires a reason for remove alone: an audit trail whose reason is
// empty is only a timestamp.
test("will not submit a remove without a reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Remove" }));
  expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
});

test("will not accept whitespace as a reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Remove" }));
  await userEvent.type(screen.getByRole("textbox"), "   ");
  expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
});

test("submits a remove once it has a reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Remove" }));
  await userEvent.type(screen.getByRole("textbox"), "Prohibited item");
  await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

  expect(mutate).toHaveBeenCalledWith(
    { listingId: LISTING_ID, action: "remove", note: "Prohibited item" },
    expect.anything(),
  );
});

test("submits a dismiss with no note at all", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Dismiss" }));
  await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

  // Omitted, not "": the API calls note optional, and an empty string is a
  // value that would land in the audit trail as one.
  expect(mutate).toHaveBeenCalledWith(
    { listingId: LISTING_ID, action: "dismiss" },
    expect.anything(),
  );
});

test("caps the note at the backend's own limit", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Remove" }));
  expect(screen.getByRole("textbox")).toHaveAttribute("maxlength", "500");
});

test("surfaces the server's own message on a conflict", () => {
  setMutation({
    isError: true,
    error: { status: 409, message: "This listing was already removed" },
  });
  renderDialog();
  expect(screen.getByRole("alert")).toHaveTextContent("This listing was already removed");
});

// reports_resolved is the one number confirming the grouping did what the
// queue promised: one decision, every open report on that listing.
test("says how many reports one decision closed", () => {
  setMutation({ isSuccess: true, data: { listing: {}, reports_resolved: 3 } });
  renderDialog();
  expect(screen.getByText("Resolved 3 reports")).toBeInTheDocument();
});
