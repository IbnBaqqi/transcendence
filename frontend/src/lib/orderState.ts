// Mirrors the rules in backend/internal/service/order.go so we don't render a
// button that will 409. Not a security check - the server still decides.
import type { Order, OrderStatus } from "../api/types";

export type OrderRole = "buyer" | "seller" | "none";
export type OrderAction = "confirm" | "handover" | "receive" | "cancel";

export interface OrderView {
  role: OrderRole;
  statusLabel: string;
  waitingOn: "you" | "them" | null;
  waitingLabel: string | null;
  actions: OrderAction[];
}

// Record<OrderStatus, string> makes a new status a compile error, not a blank pill.
const STATUS_LABELS: Record<OrderStatus, string> = {
  pending: "Pending",
  confirmed: "Confirmed",
  completed: "Completed",
  cancelled: "Cancelled",
};

export function deriveOrderView(order: Order, userId: string | undefined): OrderView {
  // userId is undefined while AuthProvider restores the session.
  const role: OrderRole =
    userId === order.buyer_id ? "buyer" : userId === order.seller_id ? "seller" : "none";

  const sellerMarked = order.seller_handed_over_at !== null;
  const buyerMarked = order.buyer_received_at !== null;
  // order.go:193 - once either side marks, cancelling is locked out.
  const handshakeStarted = sellerMarked || buyerMarked;

  const actions: OrderAction[] = [];
  let waitingOn: OrderView["waitingOn"] = null;
  let waitingLabel: string | null = null;

  if (order.status === "pending") {
    if (role === "seller") actions.push("confirm");
    if (role !== "none") actions.push("cancel");

    waitingOn = role === "seller" ? "you" : "them";
    waitingLabel =
      role === "seller"
        ? "Waiting for you to confirm this reservation."
        : "Waiting for the seller to confirm this reservation.";
  }

  if (order.status === "confirmed") {
    // Marking twice is a 409, so the button goes once that side's stamp is set.
    if (role === "seller" && !sellerMarked) actions.push("handover");
    if (role === "buyer" && !buyerMarked) actions.push("receive");
    if (role !== "none" && !handshakeStarted) actions.push("cancel");

    const yourTurn = role === "seller" ? !sellerMarked : !buyerMarked;
    if (role !== "none") waitingOn = yourTurn ? "you" : "them";

    if (!handshakeStarted) {
      waitingLabel = "Arrange the handover between yourselves, then mark it here.";
    } else if (role === "seller") {
      waitingLabel = sellerMarked
        ? "You marked the handover - waiting for the buyer to confirm receipt."
        : "The buyer confirmed receipt - mark the handover to complete the order.";
    } else if (role === "buyer") {
      waitingLabel = buyerMarked
        ? "You confirmed receipt - waiting for the seller to mark the handover."
        : "The seller marked the handover - confirm you received the goods.";
    }
  }

  // completed and cancelled are terminal: no actions, nobody waiting.

  return {
    role,
    statusLabel: STATUS_LABELS[order.status],
    waitingOn,
    waitingLabel,
    actions,
  };
}
