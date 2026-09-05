import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { OrderStatusPill } from "./OrderStatusPill";
import { ResolveOrderDialog } from "./ResolveOrderDialog";
import type { AdminOrder } from "../../api/types";

// Exactly one mark set is what "trapped mid-handover" looks like, so saying
// which side acted tells the admin who the order is waiting on.
function handshake(order: AdminOrder) {
  const seller = order.seller_handed_over_at !== null;
  const buyer = order.buyer_received_at !== null;

  if (seller && buyer) return "both";
  if (seller) return "seller";
  if (buyer) return "buyer";
  return "neither";
}

export function AdminOrderRow({ order }: { order: AdminOrder }) {
  const { t } = useTranslation();

  return (
    <li className="border-line bg-surface rounded-lg border p-3">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <Link
            to={`/listings/${order.listing_id}`}
            className="text-foreground font-medium hover:underline"
          >
            {order.listing_title}
          </Link>
          <p className="text-muted mt-1 text-sm">
            {t("adminOrders.quantityAndTotal", {
              quantity: order.quantity,
              total: order.total_price,
            })}
          </p>
          {/* Ids, not names: AdminOrder carries no usernames and a profile
              fetch per row would be an N+1 on every page. */}
          <p className="text-muted mt-1 text-sm">
            <Link to={`/users/${order.buyer_id}`} className="hover:underline">
              {t("adminOrders.buyer")}
            </Link>
            {" · "}
            <Link to={`/users/${order.seller_id}`} className="hover:underline">
              {t("adminOrders.seller")}
            </Link>
          </p>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1 text-sm">
          <OrderStatusPill status={order.status} label={t(`orders.status.${order.status}`)} />
          {/* The reason this screen exists: no party can move this one. */}
          {order.stuck && (
            <span className="text-danger text-xs font-medium">{t("adminOrders.stuck")}</span>
          )}
          <span className="text-muted text-xs">
            {t(`adminOrders.handshake.${handshake(order)}`)}
          </span>
          <span className="text-muted text-xs">
            {t("adminOrders.placed", {
              date: new Date(order.created_at).toLocaleDateString(),
            })}
          </span>
        </div>
      </div>

      <ResolveOrderDialog order={order} />
    </li>
  );
}
