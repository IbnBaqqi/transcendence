import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { MessageThread } from "../components/chat/MessageThread";
import {
  useAcceptConversation,
  useConversation,
  useDeclineConversation,
  useMarkConversationRead,
  useMessages,
  useSendMessage,
} from "../api/conversations";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { makeConversation, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Conversation, Message, User } from "../api/types";

vi.mock("../api/conversations", () => ({
  useConversation: vi.fn(),
  useMessages: vi.fn(),
  useSendMessage: vi.fn(),
  useAcceptConversation: vi.fn(),
  useDeclineConversation: vi.fn(),
  useMarkConversationRead: vi.fn(),
}));

const calls = {
  send: vi.fn(),
  accept: vi.fn(),
  decline: vi.fn(),
  markRead: vi.fn(),
};

// Generic so each mock infers its own result type: send returns a Message,
// accept/decline a Conversation, markRead nothing.
const mutation = <T,>(fn: unknown) =>
  ({ mutateAsync: fn, mutate: fn, isPending: false }) as unknown as T;

beforeEach(() => {
  Object.values(calls).forEach((fn) => fn.mockReset().mockResolvedValue(undefined));
  vi.mocked(useSendMessage).mockReturnValue(mutation(calls.send));
  vi.mocked(useAcceptConversation).mockReturnValue(mutation(calls.accept));
  vi.mocked(useDeclineConversation).mockReturnValue(mutation(calls.decline));
  vi.mocked(useMarkConversationRead).mockReturnValue(mutation(calls.markRead));
});

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "m1",
    conversation_id: "0b8e3a25-6d4c-4f99-8a32-7c1e9b2f5d68",
    sender_id: BUYER_ID,
    body: "Still available?",
    created_at: "1970-01-01T00:00:00Z",
    ...overrides,
  };
}

const VIEWER: User = { id: BUYER_ID, username: "buyer", email: "b@x.test", role: "user" };

function renderThread(conversation: Conversation, messages: Message[] = [], viewer = VIEWER) {
  vi.mocked(useConversation).mockReturnValue({
    data: conversation,
    isPending: false,
    isError: false,
  } as ReturnType<typeof useConversation>);
  vi.mocked(useMessages).mockReturnValue({ data: messages } as ReturnType<typeof useMessages>);

  const auth: AuthContextValue = {
    user: viewer,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <AuthContext.Provider value={auth}>
      <MessageThread conversationId={conversation.id} onBack={vi.fn()} />
    </AuthContext.Provider>,
  );
}

describe("MessageThread", () => {
  test("sends the typed message and clears the box", async () => {
    const user = userEvent.setup();
    renderThread(makeConversation({ status: "accepted" }));

    await user.type(screen.getByLabelText("Message"), "Yes, 2kg left");
    await user.click(screen.getByRole("button", { name: "Send" }));

    expect(calls.send).toHaveBeenCalledWith("Yes, 2kg left");
    expect(screen.getByLabelText("Message")).toHaveValue("");
  });

  test("won't send whitespace", async () => {
    const user = userEvent.setup();
    renderThread(makeConversation({ status: "accepted" }));

    await user.type(screen.getByLabelText("Message"), "   ");

    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    expect(calls.send).not.toHaveBeenCalled();
  });

  // The consent gate: a pending thread has no send box at all, for either side.
  test("a pending thread explains itself instead of offering a send box", () => {
    renderThread(makeConversation({ status: "pending", role: "buyer" }));

    expect(screen.getByText("Waiting for the seller to accept your request.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Message")).not.toBeInTheDocument();
  });

  test("the seller of a pending thread can accept or decline", async () => {
    const user = userEvent.setup();
    const conversation = makeConversation({ status: "pending", role: "seller" });
    renderThread(conversation, [], { ...VIEWER, id: SELLER_ID });

    await user.click(screen.getByRole("button", { name: "Accept" }));

    expect(calls.accept).toHaveBeenCalledWith(conversation.id);
    expect(calls.decline).not.toHaveBeenCalled();
  });

  test("the buyer of a pending thread gets no decision buttons", () => {
    renderThread(makeConversation({ status: "pending", role: "buyer" }));
    expect(screen.queryByRole("button", { name: "Accept" })).not.toBeInTheDocument();
  });

  test("own and other messages are told apart", () => {
    renderThread(makeConversation({ status: "accepted" }), [
      makeMessage({ id: "m1", sender_id: BUYER_ID, body: "mine" }),
      makeMessage({ id: "m2", sender_id: SELLER_ID, body: "theirs" }),
    ]);

    expect(screen.getByText("mine").className).not.toBe(screen.getByText("theirs").className);
  });

  // The API sends oldest-first; anything that re-sorts here renders the thread
  // backwards, which reads as coherent until you check the timestamps.
  test("renders messages in the order the API sent them", () => {
    renderThread(makeConversation({ status: "accepted" }), [
      makeMessage({ id: "m1", body: "first" }),
      makeMessage({ id: "m2", body: "second" }),
    ]);

    const bodies = screen.getAllByText(/first|second/).map((el) => el.textContent);
    expect(bodies).toEqual(["first", "second"]);
  });

  test("opening the thread clears its unread badge", () => {
    const conversation = makeConversation({ status: "accepted" });
    renderThread(conversation);
    expect(calls.markRead).toHaveBeenCalledWith(conversation.id);
  });

  test("a rejected send shows the server's reason", async () => {
    calls.send.mockRejectedValue({ status: 409, message: "This chat request was declined" });
    const user = userEvent.setup();
    renderThread(makeConversation({ status: "accepted" }));

    await user.type(screen.getByLabelText("Message"), "hello");
    await user.click(screen.getByRole("button", { name: "Send" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("This chat request was declined");
  });
});
