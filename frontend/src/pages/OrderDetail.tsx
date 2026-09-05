import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useOrder } from "../api/orders";
import { isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { deriveOrderView } from "../lib/orderState";
import { OrderActions } from "../components/objects/OrderActions";
import { OrderStatusPill } from "../components/objects/OrderStatusPill";
import { Skeleton } from "../components/objects/Skeleton";
import Button from "../components/objects/Button";
import { useModal } from "../providers/modalContext";
import type { Timestamp } from "../api/types";
import NotFound from "./NotFound";

function formatStamp(at: Timestamp | null, notYet: string) {
  return at ? new Date(at).toLocaleString() : notYet;
}

export default function OrderDetail() {
  const { id } = useParams();
  // isLoading matters here: AuthProvider reports user as null for one render,
  // and the order request can resolve first - which renders the page with no
  // role, so no buttons and the wrong side's copy.
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useTranslation();
  const { openModal } = useModal();
  const { data: order, isLoading: orderLoading, error, refetch } = useOrder(id ?? "", user?.id);

  // 403 means "not your order" - same answer as 404, and it doesn't confirm
  // that an order with this id exists.
  if (isApiError(error) && [400, 403, 404].includes(error.status)) {
    return <NotFound />;
  }

  if (authLoading || orderLoading)
    return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  // A shared order link gives a signed-out visitor a 401, which isn't in the
  // set above - so offer the login rather than a dead error line.
  if (!user) {
    return (
      <div className="max-w-page mx-auto space-y-3 px-4 py-8">
        <p className="text-muted text-sm">{t("orders.signedOutDetail")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="max-w-page mx-auto px-4 py-8">
        <Skeleton
          variant="error"
          className="h-40 w-full"
          message={isApiError(error) ? error.message : t("orders.detailError")}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const view = deriveOrderView(order, user?.id);

  return (
    <div className="max-w-page mx-auto space-y-6 px-4 py-8">
      <div>
        <Link to="/orders" className="text-muted text-sm hover:underline">
          {t("orders.allOrders")}
        </Link>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="text-foreground text-page-title font-bold">{order.listing_title}</h1>
          <OrderStatusPill status={order.status} label={t(view.statusKey)} />
        </div>
        <p className="text-muted mt-1 text-sm">
          {t("orders.yourRole", { role: t(`orders.role.${view.role}`) })}
        </p>
      </div>

      {view.waitingKey && (
        <p
          className={`text-sm ${
            view.waitingOn === "you" ? "text-foreground font-medium" : "text-muted"
          }`}
        >
          {t(view.waitingKey)}
        </p>
      )}

      <OrderActions order={order} />

      <dl className="border-line divide-line divide-y rounded-lg border">
        <Row label={t("orders.fields.quantity")} value={String(order.quantity)} />
        <Row label={t("orders.fields.unitPrice")} value={`€${order.unit_price.toFixed(2)}`} />
        <Row label={t("orders.fields.total")} value={`€${order.total_price.toFixed(2)}`} />
        <Row
          label={t("orders.fields.reserved")}
          value={new Date(order.created_at).toLocaleString()}
        />
        <Row
          label={t("orders.fields.handedOver")}
          value={formatStamp(order.seller_handed_over_at, t("orders.notYet"))}
        />
        <Row
          label={t("orders.fields.received")}
          value={formatStamp(order.buyer_received_at, t("orders.notYet"))}
        />
      </dl>

      <p className="text-muted text-sm">
        {t("orders.noPayments")}{" "}
        {/* The seller can delete a listing once nothing is in flight, and the
            order outlives it - everything above comes from its own snapshot. */}
        {order.listing_id ? (
          <Link to={`/listings/${order.listing_id}`} className="text-accent hover:underline">
            {t("orders.viewListing")}
          </Link>
        ) : (
          <span>{t("orders.listingDeleted")}</span>
        )}
      </p>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 px-4 py-2 text-sm">
      <dt className="text-muted">{label}</dt>
      <dd className="text-foreground">{value}</dd>
    </div>
  );
}
