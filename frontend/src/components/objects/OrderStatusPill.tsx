import type { OrderStatus } from "../../api/types";

// Record<OrderStatus, string> makes a new status a compile error, not a blank pill.
const PILL_STYLES: Record<OrderStatus, string> = {
  pending: "bg-surface-muted text-muted",
  confirmed: "bg-soft text-soft-contrast",
  completed: "bg-accent text-accent-contrast",
  // The two terminal failures share a ground and differ in weight: cancelled
  // means the trade never happened, refunded means it did and was unwound,
  // which is the more consequential of the two.
  cancelled: "bg-surface-soft text-muted",
  refunded: "bg-surface-soft text-foreground font-semibold",
};

export function OrderStatusPill({ status, label }: { status: OrderStatus; label: string }) {
  return (
    <span
      className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${PILL_STYLES[status]}`}
    >
      {label}
    </span>
  );
}
