import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import userEvent from "@testing-library/user-event";

import AppRouter from "../routes";
import { ChatRoot } from "../components/modal/ChatRoot";
import {
  useAcceptConversation,
  useConversation,
  useConversations,
  useDeclineConversation,
  useMarkConversationRead,
  useMessages,
  useSendMessage,
} from "../api/conversations";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { useModal } from "../providers/modalContext";
import { makeConversation, makeConversationListItem, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Message, User } from "../api/types";

// ChatRoot renders beside AppRouter in main.tsx and only works because they
// share one router - put ChatRoot back outside it and every profile Link in
// the popup throws "Cannot destructure property 'basename' of undefined",
// which the ErrorBoundary turns into "Something went wrong". This mounts the
// real components through that wiring so a regression fails the build.
vi.mock("../api/conversations", () => ({
  useConversations: vi.fn(),
  useConversation: vi.fn(),
  useMessages: vi.fn(),
  useSendMessage: vi.fn(),
  useAcceptConversation: vi.fn(),
  useDeclineConversation: vi.fn(),
  useMarkConversationRead: vi.fn(),
}));

// The routes render real Layout and Home, which hit the network; this test is
// about the root wiring, so the shell pages give way to the empty route.
vi.mock("../components/layout/Layout", async () => {
  const { Outlet } = await import("react-router-dom");
  return { default: () => <Outlet /> };
});
vi.mock("../pages/Home", () => ({ default: () => <div>home-mock</div> }));

const VIEWER: User = {
  id: BUYER_ID,
  username: "buyer",
  email: "b@x.test",
  role: "USER",
  has_password: true,
  providers: [],
};

const mutation = <T,>(fn: unknown) =>
  ({ mutateAsync: fn, mutate: fn, isPending: false }) as unknown as T;

function OpenChat() {
  const { openChat } = useModal();
  return (
    <button type="button" onClick={() => openChat()}>
      open chat
    </button>
  );
}

function renderRoot() {
  vi.mocked(useConversations).mockReturnValue({
    data: [makeConversationListItem({ id: "c1" })],
    isPending: false,
    isError: false,
  } as ReturnType<typeof useConversations>);
  vi.mocked(useConversation).mockReturnValue({
    data: makeConversation({ id: "c1" }),
    isPending: false,
    isError: false,
  } as ReturnType<typeof useConversation>);
  vi.mocked(useMessages).mockReturnValue({
    data: [] as Message[],
  } as ReturnType<typeof useMessages>);
  vi.mocked(useSendMessage).mockReturnValue(mutation(vi.fn()));
  vi.mocked(useAcceptConversation).mockReturnValue(mutation(vi.fn()));
  vi.mocked(useDeclineConversation).mockReturnValue(mutation(vi.fn()));
  vi.mocked(useMarkConversationRead).mockReturnValue(mutation(vi.fn()));

  const auth: AuthContextValue = {
    user: VIEWER,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <MemoryRouter initialEntries={["/"]}>
      <AuthContext.Provider value={auth}>
        <ModalProvider>
          <OpenChat />
          <AppRouter />
          <ChatRoot />
        </ModalProvider>
      </AuthContext.Provider>
    </MemoryRouter>,
  );
}

describe("ChatRoot wiring", () => {
  test("the popup renders and its profile links work while sharing the app router", async () => {
    const user = userEvent.setup();
    renderRoot();

    await user.click(screen.getByRole("button", { name: "open chat" }));

    // Conversation list: avatar + username link to the other user.
    const listLinks = screen.getAllByRole("link", { name: "View oscarroff's profile" });
    expect(listLinks).toHaveLength(2);
    for (const link of listLinks) {
      expect(link).toHaveAttribute("href", `/users/${SELLER_ID}`);
    }

    // Thread header: the same links survive navigation into a thread.
    await user.click(screen.getByRole("button", { name: /Open conversation with oscarroff/ }));
    const threadLinks = await screen.findAllByRole("link", { name: "View oscarroff's profile" });
    expect(threadLinks).toHaveLength(2);
    for (const link of threadLinks) {
      expect(link).toHaveAttribute("href", `/users/${SELLER_ID}`);
    }
  });
});
