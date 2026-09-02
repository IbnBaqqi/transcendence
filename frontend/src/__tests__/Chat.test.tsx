import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Chat } from "../components/modal/Chat";
import { useConversations } from "../api/conversations";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeConversationListItem, BUYER_ID } from "../test/factories";
import type { ConversationListItem, User } from "../api/types";

vi.mock("../api/conversations", () => ({ useConversations: vi.fn() }));

// The thread has its own test file; stubbing it keeps this one about routing
// between the list and a thread.
vi.mock("../components/chat/MessageThread", () => ({
  MessageThread: ({ conversationId }: { conversationId: string }) => (
    <div>thread:{conversationId}</div>
  ),
}));

type Query = ReturnType<typeof useConversations>;

const VIEWER: User = { id: BUYER_ID, username: "buyer", email: "b@x.test", role: "user" };

function renderChat(query: Partial<Query>, user: User | null = VIEWER) {
  vi.mocked(useConversations).mockReturnValue(query as Query);

  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <AuthContext.Provider value={auth}>
      <ModalProvider>
        <Chat />
      </ModalProvider>
    </AuthContext.Provider>,
  );
}

const items: ConversationListItem[] = [makeConversationListItem({ id: "c1" })];

describe("Chat", () => {
  test("lists conversations, then opens the one you pick", async () => {
    const user = userEvent.setup();
    renderChat({ data: items, isPending: false, isError: false });

    expect(screen.getByText("oscarroff")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /oscarroff/ }));

    expect(screen.getByText("thread:c1")).toBeInTheDocument();
  });

  test("a signed-out visitor is offered the login, and no request is made", () => {
    renderChat({ data: undefined, isPending: false, isError: false }, null);

    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    expect(useConversations).toHaveBeenCalledWith(false);
  });

  test("a failed load offers a retry", async () => {
    const user = userEvent.setup();
    const refetch = vi.fn();
    renderChat({
      data: undefined,
      isPending: false,
      isError: true,
      error: { status: 500, message: "boom" },
      refetch,
    } as unknown as Partial<Query>);

    expect(screen.getByText("boom")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(refetch).toHaveBeenCalled();
  });
});
