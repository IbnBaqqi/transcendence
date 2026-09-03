import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { NotificationBell } from "../components/objects/NotificationBell";
import { useMarkNotificationsRead, useNotifications } from "../api/notifications";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useModal } from "../providers/modalContext";
import { BUYER_ID, makeNotification } from "../test/factories";
import type { Notification, User } from "../api/types";

vi.mock("../api/notifications", async () => {
  const actual =
    await vi.importActual<typeof import("../api/notifications")>("../api/notifications");
  return {
    // unreadCount is the thing under test in the badge cases, so the real one runs.
    unreadCount: actual.unreadCount,
    useNotifications: vi.fn(),
    useMarkNotificationsRead: vi.fn(),
  };
});
vi.mock("../providers/modalContext", () => ({ useModal: vi.fn() }));

const VIEWER: User = {
  id: BUYER_ID,
  username: "tester",
  email: "t@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

const openChat = vi.fn();
const mutate = vi.fn();

beforeEach(() => {
  vi.mocked(useModal).mockReturnValue({ openChat } as unknown as ReturnType<typeof useModal>);
  vi.mocked(useMarkNotificationsRead).mockReturnValue({
    mutate,
    isPending: false,
    isError: false,
  } as unknown as ReturnType<typeof useMarkNotificationsRead>);
});

function renderBell(user: User | null, notifications: Notification[] | undefined) {
  vi.mocked(useNotifications).mockReturnValue({ data: notifications } as ReturnType<
    typeof useNotifications
  >);

  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter>
        <NotificationBell />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

// The query is disabled while signed out, and a disabled query reports
// isPending forever - so nothing here may key off a loading flag.
test("offers nothing while signed out", () => {
  renderBell(null, [makeNotification()]);
  expect(screen.queryByLabelText("Notifications")).not.toBeInTheDocument();
});

test("badges the unread count", () => {
  renderBell(VIEWER, [
    makeNotification({ id: "a" }),
    makeNotification({ id: "b" }),
    makeNotification({ id: "c", read_at: "1970-01-01T00:00:00Z" }),
  ]);
  expect(screen.getByText("2")).toBeInTheDocument();
});

test("wears no badge when everything is read", () => {
  renderBell(VIEWER, [makeNotification({ read_at: "1970-01-01T00:00:00Z" })]);
  expect(screen.getByLabelText("Notifications")).toBeInTheDocument();
  expect(screen.queryByText("1")).not.toBeInTheDocument();
});

// The server caps the list at 30, so a raw count would saturate there and
// read as a real total.
test("stops counting at nine", () => {
  renderBell(
    VIEWER,
    Array.from({ length: 12 }, (_, i) => makeNotification({ id: String(i) })),
  );
  expect(screen.getByText("9+")).toBeInTheDocument();
  expect(screen.queryByText("12")).not.toBeInTheDocument();
});

test("lists the notifications once opened", async () => {
  renderBell(VIEWER, [makeNotification({ listing_title: "Chanterelles" })]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  expect(screen.getByText("Someone ordered your Chanterelles")).toBeInTheDocument();
});

// undefined is not empty: before the first fetch lands the panel used to
// render an empty <ul> between its header and "See all".
test("says it is loading before the first fetch lands", async () => {
  renderBell(VIEWER, undefined);
  await userEvent.click(screen.getByLabelText("Notifications"));
  expect(screen.getByText("Loading…")).toBeInTheDocument();
  expect(screen.queryByText("Nothing new.")).not.toBeInTheDocument();
});

test("keeps the panel shut until it is asked for", () => {
  renderBell(VIEWER, [makeNotification({ listing_title: "Chanterelles" })]);
  expect(screen.queryByText("Someone ordered your Chanterelles")).not.toBeInTheDocument();
});

test("marks everything read on request", async () => {
  renderBell(VIEWER, [makeNotification()]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  await userEvent.click(screen.getByRole("button", { name: "Mark all read" }));
  expect(mutate).toHaveBeenCalled();
});

test("offers no mark-all-read when nothing is unread", async () => {
  renderBell(VIEWER, [makeNotification({ read_at: "1970-01-01T00:00:00Z" })]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  expect(screen.queryByRole("button", { name: "Mark all read" })).not.toBeInTheDocument();
});

test("sends an order notification to its order", async () => {
  renderBell(VIEWER, [makeNotification({ order_id: "order-1" })]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  expect(screen.getByRole("link", { name: /Someone ordered/ })).toHaveAttribute(
    "href",
    "/orders/order-1",
  );
});

// The chat panel is a modal with no URL, so this row cannot be a link - and it
// has to carry its own thread id, or it opens the inbox instead.
test("opens a chat request on its own thread", async () => {
  renderBell(VIEWER, [
    makeNotification({ kind: "chat_request", order_id: null, conversation_id: "conv-1" }),
  ]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  await userEvent.click(screen.getByRole("button", { name: /Someone messaged you/ }));
  expect(openChat).toHaveBeenCalledWith("conv-1");
});

test("closes on Escape", async () => {
  renderBell(VIEWER, [makeNotification({ listing_title: "Chanterelles" })]);
  await userEvent.click(screen.getByLabelText("Notifications"));
  await userEvent.keyboard("{Escape}");
  expect(screen.queryByText("Someone ordered your Chanterelles")).not.toBeInTheDocument();
});
