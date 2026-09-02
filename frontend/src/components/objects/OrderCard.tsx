import { Link } from "react-router-dom";

import { OrderStatusPill } from "./OrderStatusPill";
import { deriveOrderView } from "../../lib/orderState";
import { useAuth } from "../../hooks/useAuth";
import type { Order } from "../../api/types";

export function OrderCard({ order }: { order: Order }) {
  const { user } = useAuth();
  const view = deriveOrderView(order, user?.id);

  return (
    <Link
      to={`/orders/${order.id}`}
      className="focus-visible:ring-accent block rounded-lg focus:outline-none focus-visible:ring-2"
    >
      <article className="border-line bg-surface hover:border-accent rounded-lg border p-4 transition-colors">
        <div className="flex items-start justify-between gap-3">
          <h3 className="text-foreground font-semibold">{order.listing_title}</h3>
          <OrderStatusPill status={order.status} label={view.statusLabel} />
        </div>

        <p className="text-muted mt-1 text-sm">
          {order.quantity} × €{order.unit_price.toFixed(2)}
        </p>
        <p className="text-accent mt-2 font-medium">€{order.total_price.toFixed(2)}</p>

        {view.waitingLabel && (
          <p
            className={`mt-2 text-sm ${
              view.waitingOn === "you" ? "text-foreground font-medium" : "text-muted"
            }`}
          >
            {view.waitingLabel}
          </p>
        )}
      </article>
    </Link>
  );
}
