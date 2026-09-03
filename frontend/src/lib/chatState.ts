// Mirrors checkCanSend / checkCanDecide in backend/internal/service/conversation.go
// so we don't render a send box or a decision the server will reject.
import type { Conversation, ConversationStatus } from "../api/types";

export interface ThreadView {
  canSend: boolean;
  // Seller answering a pending request. The buyer never gets this.
  canDecide: boolean;
  // Translation keys, not text: this layer has no t(). Components translate,
  // the same way orderState hands back statusKey/waitingKey.
  sendDisabledKey: string | null;
  statusKey: string;
}

const STATUS_KEYS: Record<ConversationStatus, string> = {
  pending: "chat.status.pending",
  accepted: "chat.status.accepted",
  declined: "chat.status.declined",
};

// Pick rather than Conversation: this reads only two fields, and the list item
// (Omit<Conversation, "created_at">) has to work here too.
export function deriveThreadView(conversation: Pick<Conversation, "status" | "role">): ThreadView {
  const { status, role } = conversation;

  // Only an accepted thread takes messages - pending and declined both 409.
  const canSend = status === "accepted";
  const canDecide = status === "pending" && role === "seller";

  let sendDisabledKey: string | null = null;
  if (status === "pending") {
    sendDisabledKey =
      role === "seller" ? "chat.disabled.sellerAccept" : "chat.disabled.waitingSeller";
  } else if (status === "declined") {
    sendDisabledKey =
      role === "seller" ? "chat.disabled.youDeclined" : "chat.disabled.sellerDeclined";
  }

  return {
    canSend,
    canDecide,
    sendDisabledKey,
    statusKey: STATUS_KEYS[status],
  };
}
