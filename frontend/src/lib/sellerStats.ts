// Two derivations the dashboard needs, kept pure and out of the page.
import { deriveOrderView } from "./orderState";
import type { Listing, Order } from "../api/types";

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

// listings.quantity is a single mutable column - ordering decrements it and
// cancelling puts it back, so the DB has no memory of what was posted. Summing
// the live orders recovers it. Cancelled orders are excluded because their
// stock was already returned (order.go restoresStock); counting them would
// double-count.
//
// Exact only while GET /orders returns every order unpaginated. If orders are
// ever archived or paged, this quietly starts under-reporting.
export function deriveListingStats(listing: Listing, orders: Order[]): ListingStats {
  const sold = orders
    .filter((o) => o.listing_id === listing.id && o.status !== "cancelled")
    .reduce((total, o) => total + o.quantity, 0);

  return { sold, original: listing.quantity + sold };
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
