// Two derivations the dashboard needs, kept pure and out of the page.
import { deriveOrderView } from "./orderState";
import type { Listing, Order, OrderStatus } from "../api/types";

export interface ListingStats {
  sold: number;
  // What the seller originally posted: not stored anywhere, see below.
  original: number;
}

export interface UrgencyGroups {
  needsYou: Order[];
  inProgress: Order[];
  finished: Order[];
}

// Which statuses returned their stock to the listing: actionCancel
// (order.go restoresStock) and an admin refund (admin_order.go adminOutcomes).
// Record<OrderStatus, boolean> makes a sixth status a compile error rather than
// a silent miscount - which is exactly how refunded slipped through.
const RESTORED_STOCK: Record<OrderStatus, boolean> = {
  pending: false,
  confirmed: false,
  completed: false,
  cancelled: true,
  refunded: true,
};

// listings.quantity is a single mutable column - ordering decrements it and
// cancelling puts it back, so the DB has no memory of what was posted. Summing
// the live orders recovers it. Orders that gave their stock back are excluded;
// counting them would report a listing bigger than it ever was.
//
// Exact only while GET /orders returns every order unpaginated. If orders are
// ever archived or paged, this quietly starts under-reporting.
export function deriveListingStats(listing: Listing, orders: Order[]): ListingStats {
  const sold = orders
    .filter((o) => o.listing_id === listing.id && !RESTORED_STOCK[o.status])
    .reduce((total, o) => total + o.quantity, 0);

  return { sold, original: listing.quantity + sold };
}

// pending and confirmed are in flight; the rest are over. Record<OrderStatus,…>
// for the same reason as RESTORED_STOCK above: a sixth status should be a
// compile error rather than a silent miscount.
const IN_FLIGHT: Record<OrderStatus, boolean> = {
  pending: true,
  confirmed: true,
  completed: false,
  cancelled: false,
  refunded: false,
};

// Mirrors the rule the backend enforces (listing.go, CountActiveOrdersForListing)
// so the seller sees a disabled button with a reason rather than a 409 after
// the click. The server still decides.
export function hasSaleInProgress(listing: Listing, orders: Order[]): boolean {
  return orders.some((order) => order.listing_id === listing.id && IN_FLIGHT[order.status]);
}

// A dashboard answers "what do I need to do?", so orders group by whose turn it
// is rather than by status. deriveOrderView already computes that.
export function groupOrdersByUrgency(orders: Order[], userId?: string): UrgencyGroups {
  const groups: UrgencyGroups = { needsYou: [], inProgress: [], finished: [] };

  for (const order of orders) {
    const view = deriveOrderView(order, userId);

    if (view.waitingOn === null) {
      groups.finished.push(order);
    } else if (view.waitingOn === "you") {
      groups.needsYou.push(order);
    } else {
      groups.inProgress.push(order);
    }
  }

  return groups;
}
