// Mirrors backend/internal/service/order_test.go. If they disagree, this is wrong.
import { deriveOrderView } from "../lib/orderState";
import { makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";

const STRANGER = "33333333-3333-3333-3333-333333333333";

describe("deriveOrderView - role", () => {
  test("matches the viewer against both id fields", () => {
    const order = makeOrder();
    expect(deriveOrderView(order, BUYER_ID).role).toBe("buyer");
    expect(deriveOrderView(order, SELLER_ID).role).toBe("seller");
    expect(deriveOrderView(order, STRANGER).role).toBe("none");
  });

  test("an unknown viewer is 'none', not a crash", () => {
    expect(deriveOrderView(makeOrder(), undefined).role).toBe("none");
  });
});

describe("deriveOrderView - pending", () => {
  const order = makeOrder({ status: "pending" });

  test("only the seller can confirm", () => {
    expect(deriveOrderView(order, SELLER_ID).actions).toEqual(["confirm", "cancel"]);
    expect(deriveOrderView(order, BUYER_ID).actions).toEqual(["cancel"]);
  });

  test("the ball is in the seller's court", () => {
    expect(deriveOrderView(order, SELLER_ID).waitingOn).toBe("you");
    expect(deriveOrderView(order, BUYER_ID).waitingOn).toBe("them");
  });
});

describe("deriveOrderView - confirmed, neither side marked", () => {
  const order = makeOrder({ status: "confirmed" });

  test("each side gets its own half of the handshake, plus cancel", () => {
    expect(deriveOrderView(order, SELLER_ID).actions).toEqual(["handover", "cancel"]);
    expect(deriveOrderView(order, BUYER_ID).actions).toEqual(["receive", "cancel"]);
  });

  test("it is both their turns", () => {
    expect(deriveOrderView(order, SELLER_ID).waitingOn).toBe("you");
    expect(deriveOrderView(order, BUYER_ID).waitingOn).toBe("you");
  });
});

describe("deriveOrderView - confirmed, half-marked", () => {
  // Status is still "confirmed": the backend flips it only when both stamps land.
  const sellerDone = makeOrder({
    status: "confirmed",
    seller_handed_over_at: "2026-09-02T12:00:00Z",
  });

  test("the side that already marked cannot mark again", () => {
    expect(deriveOrderView(sellerDone, SELLER_ID).actions).toEqual([]);
  });

  test("the other side still has its half to do", () => {
    expect(deriveOrderView(sellerDone, BUYER_ID).actions).toEqual(["receive"]);
  });

  test("cancel disappears for BOTH sides once handover has started", () => {
    expect(deriveOrderView(sellerDone, SELLER_ID).actions).not.toContain("cancel");
    expect(deriveOrderView(sellerDone, BUYER_ID).actions).not.toContain("cancel");
  });

  test("waiting flips to the side that hasn't marked", () => {
    expect(deriveOrderView(sellerDone, SELLER_ID).waitingOn).toBe("them");
    expect(deriveOrderView(sellerDone, BUYER_ID).waitingOn).toBe("you");
  });

  test("the mirror case works too", () => {
    const buyerDone = makeOrder({
      status: "confirmed",
      buyer_received_at: "2026-09-02T12:00:00Z",
    });
    expect(deriveOrderView(buyerDone, BUYER_ID).actions).toEqual([]);
    expect(deriveOrderView(buyerDone, SELLER_ID).actions).toEqual(["handover"]);
    expect(deriveOrderView(buyerDone, SELLER_ID).waitingOn).toBe("you");
  });
});

describe("deriveOrderView - terminal states", () => {
  test.each(["completed", "cancelled"] as const)("%s offers nothing to anyone", (status) => {
    const order = makeOrder({ status });
    expect(deriveOrderView(order, BUYER_ID).actions).toEqual([]);
    expect(deriveOrderView(order, SELLER_ID).actions).toEqual([]);
    expect(deriveOrderView(order, BUYER_ID).waitingOn).toBeNull();
    expect(deriveOrderView(order, SELLER_ID).waitingOn).toBeNull();
  });
});

describe("deriveOrderView - strangers", () => {
  test.each(["pending", "confirmed", "completed", "cancelled"] as const)(
    "a stranger gets no actions on a %s order",
    (status) => {
      expect(deriveOrderView(makeOrder({ status }), STRANGER).actions).toEqual([]);
    },
  );
});

describe("deriveOrderView - status label", () => {
  test("capitalises the raw status for the pill", () => {
    expect(deriveOrderView(makeOrder({ status: "pending" }), BUYER_ID).statusLabel).toBe("Pending");
    expect(deriveOrderView(makeOrder({ status: "completed" }), BUYER_ID).statusLabel).toBe(
      "Completed",
    );
  });
});
