import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ConversationList } from "../components/chat/ConversationList";
import { ModalProvider } from "../providers/ModalProvider";
import { makeConversationListItem } from "../test/factories";
import type { ConversationListItem } from "../api/types";

// Avatar reaches for useModal even when it isn't editable, so every render
// here needs the provider.
function renderList(conversations: ConversationListItem[], onSelect = vi.fn()) {
  return render(
    <ModalProvider>
      <ConversationList conversations={conversations} onSelect={onSelect} />
    </ModalProvider>,
  );
}

describe("ConversationList", () => {
  test("an empty list explains how to start one", () => {
    renderList([]);
    expect(screen.getByText(/No conversations yet/)).toBeInTheDocument();
  });

  test("shows who, which listing, and the last thing said", () => {
    renderList([
      makeConversationListItem({
        last_message: { body: "Still available?", created_at: "1970-01-01T00:00:00Z" },
      }),
    ]);

    expect(screen.getByText("oscarroff")).toBeInTheDocument();
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
    expect(screen.getByText("Still available?")).toBeInTheDocument();
  });

  test("selecting a row reports its id", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const item = makeConversationListItem();
    renderList([item], onSelect);

    await user.click(screen.getByRole("button"));

    expect(onSelect).toHaveBeenCalledWith(item.id);
  });

  test("an unread count is shown, and absent when zero", () => {
    renderList([makeConversationListItem({ unread_count: 3 })]);
    expect(screen.getByText("3")).toBeInTheDocument();

    cleanup();
    renderList([makeConversationListItem({ unread_count: 0 })]);
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  // A pending request's preview would be the buyer's opening line, which reads
  // as a normal conversation - the status is the thing that needs saying.
  test("a pending thread shows its status instead of a message preview", () => {
    renderList([
      makeConversationListItem({
        status: "pending",
        last_message: { body: "Still available?", created_at: "1970-01-01T00:00:00Z" },
      }),
    ]);

    expect(screen.getByText("Request pending")).toBeInTheDocument();
    expect(screen.queryByText("Still available?")).not.toBeInTheDocument();
  });
});
