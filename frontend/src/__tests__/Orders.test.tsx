import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import Orders from "../pages/Orders";
import { useOrders } from "../api/orders";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";
import type { User } from "../api/types";

vi.mock("../api/orders", () => ({ useOrders: vi.fn() }));

type OrdersQuery = ReturnType<typeof useOrders>;

// The viewer is the BUYER on one order and the SELLER on the other, which is
// exactly the case the two tabs exist to separate.
const VIEWER = BUYER_ID;

const purchase = makeOrder({ id: "o-buy", listing_title: "Chanterelles" });
const sale = makeOrder({
  id: "o-sell",
  listing_title: "Cloudberries",
  buyer_id: SELLER_ID,
  seller_id: VIEWER,
});

const VIEWER_USER: User = {
  id: VIEWER,
  username: "tester",
  email: "t@example.com",
  role: "user",
};

function authStub(user: User | null): AuthContextValue {
  return {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

function renderPage(query: Partial<OrdersQuery>, user: User | null) {
  vi.mocked(useOrders).mockReturnValue(query as OrdersQuery);
  return render(
    <MemoryRouter>
      <AuthContext.Provider value={authStub(user)}>
        <ModalProvider>
          <Orders />
        </ModalProvider>
      </AuthContext.Provider>
    </MemoryRouter>,
  );
}

describe("Orders", () => {
  test("shows purchases first and keeps sales out of that tab", () => {
    renderPage({ data: [purchase, sale], isPending: false, isError: false }, VIEWER_USER);
    expect(screen.getByText("Chanterelles")).toBeInTheDocument();
    expect(screen.queryByText("Cloudberries")).not.toBeInTheDocument();
  });

  test("the selling tab shows the other side of the same list", async () => {
    const user = userEvent.setup();
    renderPage({ data: [purchase, sale], isPending: false, isError: false }, VIEWER_USER);

    await user.click(screen.getByRole("button", { name: /Selling/ }));

    expect(screen.getByText("Cloudberries")).toBeInTheDocument();
    expect(screen.queryByText("Chanterelles")).not.toBeInTheDocument();
  });

  test("counts both sides in the tab labels", () => {
    renderPage({ data: [purchase, sale], isPending: false, isError: false }, VIEWER_USER);
    expect(screen.getByRole("button", { name: "Buying (1)" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Selling (1)" })).toBeInTheDocument();
  });

  test("an empty list explains itself instead of rendering nothing", () => {
    renderPage({ data: [], isPending: false, isError: false }, VIEWER_USER);
    expect(screen.getByText("You haven't reserved anything yet.")).toBeInTheDocument();
  });

  test("a signed-out visitor is asked to log in, not shown an error", () => {
    renderPage({ data: undefined, isPending: false, isError: false }, null);
    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
  });
});
