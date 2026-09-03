import { deriveListingStats, groupOrdersByUrgency } from "../lib/sellerStats";
import { makeListing, makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";

// makeListing's id, which makeOrder already points its listing_id at.
const LISTING = makeListing({ quantity: 2 });

describe("deriveListingStats", () => {
  test("recovers what was posted from what is left plus what sold", () => {
    const stats = deriveListingStats(LISTING, [makeOrder({ quantity: 2, status: "confirmed" })]);
    expect(stats).toEqual({ sold: 2, original: 4 });
  });

  // Both of these already returned the stock to listing.quantity, so counting
  // them would report a listing bigger than it ever was.
  test.each(["cancelled", "refunded"] as const)(
    "%s returned its stock, so it is not a sale",
    (status) => {
      const stats = deriveListingStats(LISTING, [
        makeOrder({ id: "o1", quantity: 2, status: "confirmed" }),
        makeOrder({ id: "o2", quantity: 5, status }),
      ]);
      expect(stats).toEqual({ sold: 2, original: 4 });
    },
  );

  test("only counts orders for this listing", () => {
    const stats = deriveListingStats(LISTING, [
      makeOrder({ id: "o1", quantity: 2 }),
      makeOrder({ id: "o2", quantity: 9, listing_id: "some-other-listing" }),
    ]);
    expect(stats.sold).toBe(2);
  });

  test("a listing with no orders reports itself unchanged", () => {
    expect(deriveListingStats(LISTING, [])).toEqual({ sold: 0, original: 2 });
  });

  // The three that hold their stock. cancelled and refunded are covered above,
  // so between them every status in the union is asserted.
  test.each(["pending", "confirmed", "completed"] as const)("%s counts as sold", (status) => {
    expect(deriveListingStats(LISTING, [makeOrder({ quantity: 1, status })]).sold).toBe(1);
  });
});

describe("groupOrdersByUrgency", () => {
  test("a pending order is the seller's move, not the buyer's", () => {
    const order = makeOrder({ status: "pending" });

    expect(groupOrdersByUrgency([order], SELLER_ID).needsYou).toEqual([order]);
    expect(groupOrdersByUrgency([order], BUYER_ID).inProgress).toEqual([order]);
  });

  test("a half-marked handshake waits on whoever hasn't marked", () => {
    const sellerDone = makeOrder({
      status: "confirmed",
      seller_handed_over_at: "2026-09-02T12:00:00Z",
    });

    expect(groupOrdersByUrgency([sellerDone], SELLER_ID).inProgress).toEqual([sellerDone]);
    expect(groupOrdersByUrgency([sellerDone], BUYER_ID).needsYou).toEqual([sellerDone]);
  });

  test.each(["completed", "cancelled", "refunded"] as const)("%s is finished", (status) => {
    const order = makeOrder({ status });
    expect(groupOrdersByUrgency([order], SELLER_ID).finished).toEqual([order]);
  });

  test("every order lands in exactly one group", () => {
    const orders = [
      makeOrder({ id: "a", status: "pending" }),
      makeOrder({ id: "b", status: "confirmed" }),
      makeOrder({ id: "c", status: "completed" }),
      makeOrder({ id: "d", status: "cancelled" }),
    ];
    const groups = groupOrdersByUrgency(orders, SELLER_ID);

    expect(groups.needsYou.length + groups.inProgress.length + groups.finished.length).toBe(
      orders.length,
    );
  });
});
