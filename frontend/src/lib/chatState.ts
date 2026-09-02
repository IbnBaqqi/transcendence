// Mirrors checkCanSend / checkCanDecide in backend/internal/service/conversation.go
// so we don't render a send box or a decision the server will reject.
import type { Conversation, ConversationStatus } from "../api/types";

export interface ThreadView {
  canSend: boolean;
  // Seller answering a pending request. The buyer never gets this.
  canDecide: boolean;
  // Why the send box is closed, addressed to whoever is reading it.
  sendDisabledReason: string | null;
  statusLabel: string;
}

const STATUS_LABELS: Record<ConversationStatus, string> = {
  pending: "Request pending",
  accepted: "Accepted",
  declined: "Declined",
};

// Pick rather than Conversation: this reads only two fields, and the list item
// (Omit<Conversation, "created_at">) has to work here too.
export function deriveThreadView(conversation: Pick<Conversation, "status" | "role">): ThreadView {
  const { status, role } = conversation;

  // Only an accepted thread takes messages - pending and declined both 409.
  const canSend = status === "accepted";
  const canDecide = status === "pending" && role === "seller";

  let sendDisabledReason: string | null = null;
  if (status === "pending") {
    sendDisabledReason =
      role === "seller"
        ? "Accept this request to reply."
        : "Waiting for the seller to accept your request.";
  } else if (status === "declined") {
    sendDisabledReason =
      role === "seller" ? "You declined this request." : "The seller declined this request.";
  }

  return {
    canSend,
    canDecide,
    sendDisabledReason,
    statusLabel: STATUS_LABELS[status],
  };
}
