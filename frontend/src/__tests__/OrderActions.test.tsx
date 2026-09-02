import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { OrderActions } from "../components/objects/OrderActions";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useCancelOrder, useConfirmOrder, useHandoverOrder, useReceiveOrder } from "../api/orders";
import { makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Order, User } from "../api/types";

vi.mock("../api/orders", () => ({
  useConfirmOrder: vi.fn(),
  useHandoverOrder: vi.fn(),
  useReceiveOrder: vi.fn(),
  useCancelOrder: vi.fn(),
}));

// One mock per action, so a test can tell which endpoint was hit.
const calls = {
  confirm: vi.fn(),
  handover: vi.fn(),
  receive: vi.fn(),
  cancel: vi.fn(),
};

type Mutation = ReturnType<typeof useConfirmOrder>;

const stub = (mutateAsync: unknown) => ({ mutateAsync, isPending: false }) as Mutation;

beforeEach(() => {
  Object.values(calls).forEach((fn) => fn.mockReset().mockResolvedValue(undefined));
  vi.mocked(useConfirmOrder).mockReturnValue(stub(calls.confirm));
  vi.mocked(useHandoverOrder).mockReturnValue(stub(calls.handover));
  vi.mocked(useReceiveOrder).mockReturnValue(stub(calls.receive));
  vi.mocked(useCancelOrder).mockReturnValue(stub(calls.cancel));
});

function authStub(id: string): AuthContextValue {
  const user: User = { id, username: "tester", email: "t@example.com", role: "user" };
  return {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

function renderActions(order: Order, viewerId: string) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={authStub(viewerId)}>
        <OrderActions order={order} />
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("OrderActions", () => {
  test("a pending order offers the seller confirm and cancel", () => {
    renderActions(makeOrder({ status: "pending" }), SELLER_ID);
    expect(screen.getByRole("button", { name: "Confirm reservation" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel order" })).toBeInTheDocument();
  });

  test("the buyer of a pending order gets no confirm button", () => {
    renderActions(makeOrder({ status: "pending" }), BUYER_ID);
    expect(screen.queryByRole("button", { name: "Confirm reservation" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel order" })).toBeInTheDocument();
  });

  test("clicking an action calls that action's endpoint with the order id", async () => {
    const user = userEvent.setup();
    const order = makeOrder({ status: "pending" });
    renderActions(order, SELLER_ID);

    await user.click(screen.getByRole("button", { name: "Confirm reservation" }));

    expect(calls.confirm).toHaveBeenCalledWith(order.id);
    expect(calls.cancel).not.toHaveBeenCalled();
  });

  // The map in OrderActions wires four hooks to four action names. Only
  // asserting confirm leaves the others free to be crossed: the buttons would
  // read correctly and POST the wrong transition.
  test("the seller's handover button hits the handover endpoint", async () => {
    const user = userEvent.setup();
    const order = makeOrder({ status: "confirmed" });
    renderActions(order, SELLER_ID);

    await user.click(screen.getByRole("button", { name: "Mark handed over" }));

    expect(calls.handover).toHaveBeenCalledWith(order.id);
    expect(calls.receive).not.toHaveBeenCalled();
  });

  test("the buyer's receipt button hits the receive endpoint", async () => {
    const user = userEvent.setup();
    const order = makeOrder({ status: "confirmed" });
    renderActions(order, BUYER_ID);

    await user.click(screen.getByRole("button", { name: "Confirm receipt" }));

    expect(calls.receive).toHaveBeenCalledWith(order.id);
    expect(calls.handover).not.toHaveBeenCalled();
  });

  test("cancel hits the cancel endpoint", async () => {
    const user = userEvent.setup();
    const order = makeOrder({ status: "pending" });
    renderActions(order, BUYER_ID);

    await user.click(screen.getByRole("button", { name: "Cancel order" }));

    expect(calls.cancel).toHaveBeenCalledWith(order.id);
  });

  test("renders nothing at all for a finished order", () => {
    const { container } = renderActions(makeOrder({ status: "completed" }), BUYER_ID);
    expect(container).toBeEmptyDOMElement();
  });

  // The common failure: the other side acted while this page sat open.
  test("a 409 explains the order moved instead of showing the raw error", async () => {
    calls.confirm.mockRejectedValue({
      status: 409,
      message: "Cannot confirm an order that is cancelled",
    });
    const user = userEvent.setup();
    renderActions(makeOrder({ status: "pending" }), SELLER_ID);

    await user.click(screen.getByRole("button", { name: "Confirm reservation" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("This order just changed");
  });

  // A 403 means our role logic is wrong, so the backend's wording is the useful one.
  test("a 403 shows the message the backend sent", async () => {
    calls.confirm.mockRejectedValue({
      status: 403,
      message: "Only the seller can confirm this order",
    });
    const user = userEvent.setup();
    renderActions(makeOrder({ status: "pending" }), SELLER_ID);

    await user.click(screen.getByRole("button", { name: "Confirm reservation" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Only the seller can confirm this order",
    );
  });
});
