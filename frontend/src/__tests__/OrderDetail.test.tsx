import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import OrderDetail from "../pages/OrderDetail";
import {
  useOrder,
  useCancelOrder,
  useConfirmOrder,
  useHandoverOrder,
  useReceiveOrder,
} from "../api/orders";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeOrder, SELLER_ID } from "../test/factories";
import type { Order, User } from "../api/types";

// OrderDetail renders OrderActions, which reaches for all four mutation hooks.
vi.mock("../api/orders", () => ({
  useOrder: vi.fn(),
  useConfirmOrder: vi.fn(),
  useHandoverOrder: vi.fn(),
  useReceiveOrder: vi.fn(),
  useCancelOrder: vi.fn(),
}));

type Mutation = ReturnType<typeof useConfirmOrder>;

const idleMutation = () => ({ mutateAsync: vi.fn(), isPending: false }) as unknown as Mutation;

beforeEach(() => {
  vi.mocked(useConfirmOrder).mockReturnValue(idleMutation());
  vi.mocked(useHandoverOrder).mockReturnValue(idleMutation());
  vi.mocked(useReceiveOrder).mockReturnValue(idleMutation());
  vi.mocked(useCancelOrder).mockReturnValue(idleMutation());
});

const SELLER: User = {
  id: SELLER_ID,
  username: "seller",
  email: "s@example.test",
  role: "user",
  has_password: true,
  providers: [],
};

function renderDetail(
  query: { data?: Order; isLoading?: boolean; error?: unknown; refetch?: () => void },
  auth: { user: User | null; isLoading?: boolean },
) {
  vi.mocked(useOrder).mockReturnValue({
    error: null,
    ...query,
  } as ReturnType<typeof useOrder>);

  const value: AuthContextValue = {
    user: auth.user,
    isLoading: auth.isLoading ?? false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={value}>
        <ModalProvider>
          <MemoryRouter initialEntries={["/orders/o1"]}>
            <Routes>
              <Route path="/orders/:id" element={<OrderDetail />} />
            </Routes>
          </MemoryRouter>
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("OrderDetail", () => {
  test("gives the seller their own action and their own copy", () => {
    renderDetail({ data: makeOrder({ status: "pending" }), isLoading: false }, { user: SELLER });

    expect(screen.getByRole("button", { name: "Confirm reservation" })).toBeInTheDocument();
    expect(screen.getByText("Waiting for you to confirm this reservation.")).toBeInTheDocument();
  });

  // The regression: the order request carries the localStorage token, so it can
  // resolve while AuthProvider is still asking /auth/me. Rendering then means a
  // seller sees no buttons and the buyer's sentence.
  test("waits out the session restore instead of rendering with no role", () => {
    renderDetail(
      { data: makeOrder({ status: "pending" }), isLoading: false },
      { user: null, isLoading: true },
    );

    expect(screen.getByText("Loading…")).toBeInTheDocument();
    expect(
      screen.queryByText("Waiting for the seller to confirm this reservation."),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  test("shows 'Not yet' for handshake steps that haven't happened", () => {
    renderDetail({ data: makeOrder({ status: "confirmed" }), isLoading: false }, { user: SELLER });

    // Seller handed over + buyer confirmed receipt, neither marked.
    expect(screen.getAllByText("Not yet")).toHaveLength(2);
  });

  test.each([400, 403, 404])("a %s renders the not-found page", (status) => {
    renderDetail(
      { data: undefined, isLoading: false, error: { status, message: "nope" } },
      { user: SELLER },
    );

    expect(screen.getByText("404 - Page not found")).toBeInTheDocument();
  });

  test("any other error offers a retry rather than a dead end", async () => {
    const user = userEvent.setup();
    const refetch = vi.fn();
    renderDetail(
      {
        data: undefined,
        isLoading: false,
        error: { status: 500, message: "Server exploded" },
        refetch,
      },
      { user: SELLER },
    );

    expect(screen.getByText("Server exploded")).toBeInTheDocument();
    expect(screen.queryByText("404 - Page not found")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(refetch).toHaveBeenCalled();
  });

  // A shared order link: the 401 isn't in the not-found set, so without this
  // branch the visitor gets a bare error line and no way to act.
  test("a signed-out visitor following an order link is offered the login", () => {
    renderDetail(
      {
        data: undefined,
        isLoading: false,
        error: { status: 401, message: "Authentication required" },
      },
      { user: null },
    );

    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    // No user id means the query stays disabled: fetching would 401 and burn a
    // refresh, the same reason useOrders takes `enabled`.
    expect(useOrder).toHaveBeenCalledWith("o1", undefined);
  });

  test("passes the viewer's id to the query so it only runs when signed in", () => {
    renderDetail({ data: makeOrder(), isLoading: false }, { user: SELLER });
    expect(useOrder).toHaveBeenCalledWith("o1", SELLER_ID);
  });
});
