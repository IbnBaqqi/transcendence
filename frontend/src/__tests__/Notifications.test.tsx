import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import Notifications from "../pages/Notifications";
import { useMarkNotificationsRead, useNotifications } from "../api/notifications";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useModal } from "../providers/modalContext";
import { BUYER_ID, makeNotification } from "../test/factories";
import type { User } from "../api/types";

vi.mock("../api/notifications", async () => {
  const actual =
    await vi.importActual<typeof import("../api/notifications")>("../api/notifications");
  return {
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
  role: "user",
  has_password: true,
  providers: [],
};

beforeEach(() => {
  vi.mocked(useModal).mockReturnValue({
    openModal: vi.fn(),
    openChat: vi.fn(),
  } as unknown as ReturnType<typeof useModal>);
  vi.mocked(useMarkNotificationsRead).mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
  } as unknown as ReturnType<typeof useMarkNotificationsRead>);
});

function renderPage(user: User | null, query: Partial<ReturnType<typeof useNotifications>>) {
  vi.mocked(useNotifications).mockReturnValue(query as ReturnType<typeof useNotifications>);

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
        <Notifications />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

test("asks a signed-out visitor to sign in", () => {
  renderPage(null, { isPending: true });
  expect(screen.getByText("Sign in to see your notifications.")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
});

test("says so when there is nothing", () => {
  renderPage(VIEWER, { data: [], isPending: false, isError: false });
  expect(screen.getByText("Nothing new.")).toBeInTheDocument();
});

test("lists what there is", () => {
  renderPage(VIEWER, {
    data: [makeNotification({ listing_title: "Chanterelles" })],
    isPending: false,
    isError: false,
  });
  expect(screen.getByText("Someone ordered your Chanterelles")).toBeInTheDocument();
});

test("offers a retry when the list will not load", async () => {
  const refetch = vi.fn();
  renderPage(VIEWER, {
    data: undefined,
    isPending: false,
    isError: true,
    error: new Error("boom"),
    refetch,
  });
  expect(screen.getByText("Couldn't load your notifications.")).toBeInTheDocument();

  await userEvent.click(screen.getByRole("button", { name: "Try again" }));
  expect(refetch).toHaveBeenCalled();
});

test("offers mark-all-read only while something is unread", () => {
  renderPage(VIEWER, {
    data: [makeNotification({ read_at: "1970-01-01T00:00:00Z" })],
    isPending: false,
    isError: false,
  });
  expect(screen.queryByRole("button", { name: "Mark all read" })).not.toBeInTheDocument();
});
