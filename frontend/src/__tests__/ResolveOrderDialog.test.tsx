import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ResolveOrderDialog } from "../components/objects/ResolveOrderDialog";
import { useResolveOrder } from "../api/adminOrders";
import type { AdminOrder } from "../api/types";

vi.mock("../api/adminOrders", () => ({ useResolveOrder: vi.fn() }));

const ORDER_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";
const mutate = vi.fn();

function setMutation(over: Record<string, unknown> = {}) {
  vi.mocked(useResolveOrder).mockReturnValue({
    mutate,
    isPending: false,
    isError: false,
    error: null,
    ...over,
  } as unknown as ReturnType<typeof useResolveOrder>);
}

beforeEach(() => setMutation());

function makeOrder(over: Partial<AdminOrder> = {}): AdminOrder {
  return {
    id: ORDER_ID,
    listing_id: "01a02305-b81c-7dcb-86a0-7f75e33e0af4",
    listing_title: "Golden Chanterelles",
    buyer_id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
    seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    quantity: 2,
    unit_price: 18.5,
    total_price: 37,
    status: "confirmed",
    // Exactly one mark: the handshake-stuck shape, which takes all three.
    seller_handed_over_at: "2026-08-02T00:00:00Z",
    buyer_received_at: null,
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T00:00:00Z",
    stuck: true,
    ...over,
  };
}

// Neither mark, and stuck: the stranded shape. The server takes cancelled
// alone there, so offering the others would generate 409s deliberately.
const stranded = () =>
  makeOrder({ seller_handed_over_at: null, buyer_received_at: null, status: "pending" });

function renderDialog(order = makeOrder()) {
  render(<ResolveOrderDialog order={order} />);
}

// Only a stuck order can be resolved at all.
test("offers nothing on an order somebody can still move", () => {
  renderDialog(makeOrder({ stuck: false }));
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});

test("offers all three outcomes when one side has acted", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  for (const name of ["Completed", "Cancelled", "Refunded"]) {
    expect(screen.getByRole("button", { name })).toBeInTheDocument();
  }
});

// completed would assert a handover that never happened, and refunded implies
// a trade that got far enough to unwind.
test("offers only cancelled when neither side ever acted", async () => {
  renderDialog(stranded());
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  expect(screen.getByRole("button", { name: "Cancelled" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Completed" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Refunded" })).not.toBeInTheDocument();
});

// The other handshake-stuck shape: the buyer received but the seller never
// marked a handover. Exactly one mark either way is stuck, not stranded, so
// all three outcomes stay legal - a condition that only reads the seller's
// mark would wrongly restrict this to cancelled.
test("offers all three when only the buyer has acted", async () => {
  renderDialog(
    makeOrder({ seller_handed_over_at: null, buyer_received_at: "2026-08-02T00:00:00Z" }),
  );
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  for (const name of ["Completed", "Cancelled", "Refunded"]) {
    expect(screen.getByRole("button", { name })).toBeInTheDocument();
  }
});

test("says why only cancelled is on offer", async () => {
  renderDialog(stranded());
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  expect(
    screen.getByText("Neither side ever acted, so this can only be cancelled."),
  ).toBeInTheDocument();
});

test("will not resolve without an outcome", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  await userEvent.type(screen.getByRole("textbox"), "Buyer never collected");
  expect(screen.getByRole("button", { name: "Resolve order" })).toBeDisabled();
});

test("will not resolve without a reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  await userEvent.click(screen.getByRole("button", { name: "Cancelled" }));
  expect(screen.getByRole("button", { name: "Resolve order" })).toBeDisabled();
});

test("will not accept whitespace as a reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  await userEvent.click(screen.getByRole("button", { name: "Cancelled" }));
  await userEvent.type(screen.getByRole("textbox"), "   ");
  expect(screen.getByRole("button", { name: "Resolve order" })).toBeDisabled();
});

test("resolves with the outcome and the trimmed reason", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  await userEvent.click(screen.getByRole("button", { name: "Refunded" }));
  await userEvent.type(screen.getByRole("textbox"), "  Seller kept the goods  ");
  await userEvent.click(screen.getByRole("button", { name: "Resolve order" }));

  expect(mutate).toHaveBeenCalledWith(
    { orderId: ORDER_ID, outcome: "refunded", reason: "Seller kept the goods" },
    expect.anything(),
  );
});

// The reason lands in the order's history and both parties read it there.
test("says who the reason is written for", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  expect(screen.getByLabelText("Reason (both parties can read this)")).toBeInTheDocument();
});

test("caps the reason at the backend's own limit", async () => {
  renderDialog();
  await userEvent.click(screen.getByRole("button", { name: "Resolve" }));
  expect(screen.getByRole("textbox")).toHaveAttribute("maxlength", "500");
});

// A 409 is "not stuck any more" or "that shape only takes cancelled" - only
// the server's message tells them apart, so it has to reach the screen.
test("surfaces the server's own message on a conflict", () => {
  setMutation({
    isError: true,
    error: {
      status: 409,
      message: "An order neither party ever acted on can only be resolved as cancelled",
    },
  });
  renderDialog();
  expect(screen.getByRole("alert")).toHaveTextContent("can only be resolved as cancelled");
});
