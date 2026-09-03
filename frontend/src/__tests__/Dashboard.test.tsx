import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import Dashboard from "../pages/Dashboard";
import { useOrders } from "../api/orders";
import { useSearchListings } from "../api/listings";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeListing, makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Listing, Order, User } from "../api/types";

vi.mock("../api/orders", () => ({ useOrders: vi.fn() }));
vi.mock("../api/listings", () => ({ useSearchListings: vi.fn() }));

const SELLER: User = {
  id: SELLER_ID,
  username: "seller",
  email: "s@x.test",
  role: "USER",
  has_password: true,
  providers: [],
};

function renderDashboard({
  listings = [] as Listing[],
  orders = [] as Order[],
  user = SELLER as User | null,
  listingsState = {},
  ordersState = {},
} = {}) {
  vi.mocked(useSearchListings).mockReturnValue({
    data: { items: listings, total: listings.length, page: 1, limit: 50, total_pages: 1 },
    isPending: false,
    isError: false,
    ...listingsState,
  } as unknown as ReturnType<typeof useSearchListings>);

  vi.mocked(useOrders).mockReturnValue({
    data: orders,
    isPending: false,
    isError: false,
    ...ordersState,
  } as unknown as ReturnType<typeof useOrders>);

  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <MemoryRouter>
      <AuthContext.Provider value={auth}>
        <ModalProvider>
          <Dashboard />
        </ModalProvider>
      </AuthContext.Provider>
    </MemoryRouter>,
  );
}

describe("Dashboard", () => {
  test("leads with the orders waiting on the seller", () => {
    renderDashboard({ orders: [makeOrder({ status: "pending" })] });
    expect(screen.getByText("Needs you (1)")).toBeInTheDocument();
  });

  // GET /orders carries this user's purchases too; those belong on /orders.
  test("a purchase does not appear on the seller's dashboard", () => {
    renderDashboard({
      orders: [makeOrder({ id: "mine", buyer_id: SELLER_ID, seller_id: BUYER_ID })],
    });
    expect(screen.queryByText(/Needs you/)).not.toBeInTheDocument();
    expect(screen.getByText("No one has ordered from you yet.")).toBeInTheDocument();
  });

  test("a listing shows what sold against what was posted", () => {
    renderDashboard({
      listings: [makeListing({ quantity: 2 })],
      orders: [makeOrder({ quantity: 2, status: "confirmed" })],
    });
    expect(screen.getByText(/2 of 4 kg left/)).toBeInTheDocument();
    expect(screen.getByText(/2 sold/)).toBeInTheDocument();
  });

  test("a seller with nothing listed is pointed at the add page", () => {
    renderDashboard();
    expect(screen.getByText("You haven't posted a listing yet.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Post your first listing" })).toHaveAttribute(
      "href",
      "/addlisting",
    );
  });

  test("finished orders are separated from live ones", () => {
    renderDashboard({
      orders: [
        makeOrder({ id: "a", status: "pending" }),
        makeOrder({ id: "b", status: "completed" }),
      ],
    });
    expect(screen.getByText("Needs you (1)")).toBeInTheDocument();
    expect(screen.getByText("Finished (1)")).toBeInTheDocument();
  });

  // limit is 50; without this a seller with more just sees the first page.
  test("says so when the listing count is truncated", () => {
    renderDashboard({
      listings: [makeListing()],
      listingsState: {
        data: { items: [makeListing()], total: 51, page: 1, limit: 50, total_pages: 2 },
      },
    });
    expect(screen.getByText("Showing 1 of 51")).toBeInTheDocument();
  });

  test("says nothing when everything fits", () => {
    renderDashboard({ listings: [makeListing()] });
    expect(screen.queryByText(/Showing/)).not.toBeInTheDocument();
  });

  test("a signed-out visitor is offered the login and neither query fires", () => {
    renderDashboard({ user: null });

    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    expect(useOrders).toHaveBeenCalledWith({ enabled: false });
    expect(useSearchListings).toHaveBeenCalledWith(
      { seller_id: undefined, limit: 50 },
      { enabled: false },
    );
  });
});
