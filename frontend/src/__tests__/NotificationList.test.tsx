import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { NotificationList } from "../components/objects/NotificationList";
import { useModal } from "../providers/modalContext";
import { makeNotification } from "../test/factories";
import type { Notification } from "../api/types";

vi.mock("../providers/modalContext", () => ({ useModal: vi.fn() }));

const openChat = vi.fn();

beforeEach(() => {
  vi.mocked(useModal).mockReturnValue({
    openModal: vi.fn(),
    openChat,
  } as unknown as ReturnType<typeof useModal>);
});

function renderList(notification: Partial<Notification>) {
  render(
    <MemoryRouter>
      <NotificationList notifications={[makeNotification(notification)]} />
    </MemoryRouter>,
  );
}

const ORDER = "0a7d2f14-5c3b-4e88-9f21-6b0d8a1e4c57";
const CONVERSATION = "0b8e3a25-6d4c-4f99-8a32-7c1e9b2f5d68";

test.each([
  ["order_placed", "Someone ordered your Golden Chanterelles"],
  ["order_confirmed", "Your Golden Chanterelles order was confirmed"],
  ["order_completed", "Your Golden Chanterelles order is complete"],
])("%s reads its own sentence and links to its order", (kind, text) => {
  renderList({ kind, order_id: ORDER, conversation_id: null });

  const row = screen.getByRole("link", { name: text });
  expect(row).toHaveAttribute("href", `/orders/${ORDER}`);
});

test("a chat request opens the panel rather than navigating", async () => {
  const user = userEvent.setup();
  renderList({
    kind: "chat_request",
    order_id: null,
    conversation_id: CONVERSATION,
    listing_title: "Golden Chanterelles",
  });

  await user.click(screen.getByRole("button"));

  expect(openChat).toHaveBeenCalledWith(CONVERSATION);
});

// The bug this replaces. A kind carrying neither an order nor a conversation
// fell through to openChat(undefined): the panel opened on nothing, with no
// error and no empty state. The backend can ship a kind before this build
// knows it, so an unknown one has to render rather than guess.
// The kind here has to be one the build genuinely does not know. It was
// new_follower until that shipped, which is the point: this test goes stale
// every time a kind lands, and pointing it at a real one tests nothing.
test("an unrecognised kind renders as text and opens nothing", async () => {
  const user = userEvent.setup();
  renderList({
    kind: "listing_price_dropped",
    order_id: null,
    conversation_id: null,
    actor_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    listing_title: null,
  });

  expect(screen.getByText("Something happened on your account")).toBeInTheDocument();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();

  await user.click(screen.getByText("Something happened on your account"));
  expect(openChat).not.toHaveBeenCalled();
});

// A known kind whose subject is missing is the same problem wearing a
// recognised name: route on the kind, but only when the id it needs is there.
test("a known kind with no subject id goes nowhere", () => {
  renderList({ kind: "order_placed", order_id: null, conversation_id: null });

  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});

// listing_title is nullable since the notification subjects migration. It must
// read as an absence, not as the word "null".
test("a row with no listing title does not render the word null", () => {
  renderList({ kind: "order_placed", order_id: ORDER, listing_title: null });

  expect(screen.queryByText(/null/i)).not.toBeInTheDocument();
});

const ACTOR = "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34";
const LISTING = "5d2b9e37-1a4c-4b62-8e73-9f0a2c5d7b81";

test.each([
  ["new_follower", "You have a new follower"],
  ["review_received", "You got a new review"],
  ["saved_listing_deleted", "The Golden Chanterelles you saved was deleted"],
])("%s links to the person it points at", (kind, text) => {
  renderList({ kind, order_id: null, conversation_id: null, actor_id: ACTOR });

  expect(screen.getByRole("link", { name: text })).toHaveAttribute("href", `/users/${ACTOR}`);
});

test.each([
  ["listing_removed", "A moderator removed your Golden Chanterelles"],
  ["saved_listing_gone", "The Golden Chanterelles you saved has sold out"],
])("%s links to the listing it points at", (kind, text) => {
  renderList({ kind, order_id: null, conversation_id: null, listing_id: LISTING });

  expect(screen.getByRole("link", { name: text })).toHaveAttribute("href", `/listings/${LISTING}`);
});

// The two kinds whose rows carry no listing_title: interpolating one would
// leave a gap where a name should be.
test.each(["new_follower", "review_received"])("%s reads without a listing title", (kind) => {
  renderList({ kind, order_id: null, conversation_id: null, actor_id: ACTOR, listing_title: null });

  const row = screen.getByRole("link");
  expect(row.textContent).not.toMatch(/\s{2,}|undefined|null/);
});
