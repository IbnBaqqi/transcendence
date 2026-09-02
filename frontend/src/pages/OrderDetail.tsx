import { Link, useParams } from "react-router-dom";

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

function formatStamp(at: Timestamp | null) {
  return at ? new Date(at).toLocaleString() : "Not yet";
}

export default function OrderDetail() {
  const { id } = useParams();
  // isLoading matters here: AuthProvider reports user as null for one render,
  // and the order request can resolve first - which renders the page with no
  // role, so no buttons and the wrong side's copy.
  const { user, isLoading: authLoading } = useAuth();
  const { openModal } = useModal();
  const { data: order, isLoading: orderLoading, error, refetch } = useOrder(id ?? "");

  // 403 means "not your order" - same answer as 404, and it doesn't confirm
  // that an order with this id exists.
  if (isApiError(error) && [400, 403, 404].includes(error.status)) {
    return <NotFound />;
  }

  if (authLoading || orderLoading) return <p className="text-muted p-8 text-sm">Loading…</p>;

  // A shared order link gives a signed-out visitor a 401, which isn't in the
  // set above - so offer the login rather than a dead error line.
  if (!user) {
    return (
      <div className="mx-auto max-w-2xl space-y-3 px-4 py-8">
        <p className="text-muted text-sm">You're signed out. Log in to see this order.</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          Log In
        </Button>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Skeleton
          variant="error"
          className="h-40 w-full"
          message={isApiError(error) ? error.message : "Couldn't load this order."}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const view = deriveOrderView(order, user?.id);

  return (
    <div className="mx-auto max-w-2xl space-y-6 px-4 py-8">
      <div>
        <Link to="/orders" className="text-muted text-sm hover:underline">
          ← All orders
        </Link>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="text-foreground text-2xl font-bold">{order.listing_title}</h1>
          <OrderStatusPill status={order.status} label={view.statusLabel} />
        </div>
        <p className="text-muted mt-1 text-sm">
          You're the {view.role === "none" ? "—" : view.role} on this order.
        </p>
      </div>

      {view.waitingLabel && (
        <p
          className={`text-sm ${
            view.waitingOn === "you" ? "text-foreground font-medium" : "text-muted"
          }`}
        >
          {view.waitingLabel}
        </p>
      )}

      <OrderActions order={order} />

      <dl className="border-line divide-line divide-y rounded-lg border">
        <Row label="Quantity" value={String(order.quantity)} />
        <Row label="Unit price" value={`€${order.unit_price.toFixed(2)}`} />
        <Row label="Total" value={`€${order.total_price.toFixed(2)}`} />
        <Row label="Reserved" value={new Date(order.created_at).toLocaleString()} />
        <Row label="Seller handed over" value={formatStamp(order.seller_handed_over_at)} />
        <Row label="Buyer confirmed receipt" value={formatStamp(order.buyer_received_at)} />
      </dl>

      <p className="text-muted text-sm">
        Metsätori handles no payments - arrange payment and pickup between yourselves.{" "}
        <Link to={`/listings/${order.listing_id}`} className="text-accent hover:underline">
          View the listing
        </Link>
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
