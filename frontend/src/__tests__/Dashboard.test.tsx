import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import Dashboard from "../pages/Dashboard";
import { useOrders } from "../api/orders";
import { useSearchListings } from "../api/listings";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeListing, makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Listing, Order, User } from "../api/types";

vi.mock("../api/orders", () => ({ useOrders: vi.fn() }));
// The rows carry a delete action now, so the module needs its mutation too.
vi.mock("../api/listings", () => ({
  useSearchListings: vi.fn(),
  useDeleteListing: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

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
    data: { items: listings, total: listings.length, page: 1, limit: 20, total_pages: 1 },
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
  // The dashboard is the one caller entitled to sold-out rows, and without
  // asking for them a listing vanishes from its owner's inventory the moment it
  // sells out - which is the row they most need, since it is the one to restock
  // or delist. SellerListingRow's "Sold out" state is unreachable without this.
  test("asks for the seller's sold-out listings, not just the sellable ones", () => {
    renderDashboard();

    expect(vi.mocked(useSearchListings)).toHaveBeenCalledWith(
      expect.objectContaining({ seller_id: SELLER_ID, include_sold_out: true }),
      expect.anything(),
    );
  });

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
    expect(screen.getByText("Orders (1)")).toBeInTheDocument();
  });

  // A seller with more than one page's worth used to see the first 50 and a
  // line saying so. The rest are reachable now.
  test("offers a pager when the inventory is longer than a page", () => {
    renderDashboard({
      listings: [makeListing()],
      listingsState: {
        data: { items: [makeListing()], total: 65, page: 1, limit: 20, total_pages: 4 },
      },
    });

    expect(screen.getByText("Page 1 of 4")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });

  test("asks for the next page when the pager is used", async () => {
    const user = userEvent.setup();
    renderDashboard({
      listings: [makeListing()],
      listingsState: {
        data: { items: [makeListing()], total: 65, page: 1, limit: 20, total_pages: 4 },
      },
    });

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(vi.mocked(useSearchListings)).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2, include_sold_out: true }),
      expect.anything(),
    );
  });

  // A stale or shared ?page=N link outlives the listings it pointed at, and a
  // moderator removing enough of them shrinks the set underneath the seller.
  // Keying the empty state on this page's length rather than on the total told
  // a seller with 30 listings they had none, and removed the pager that is the
  // only way back.
  test("keeps the pager on a page past the end instead of claiming there is nothing", () => {
    renderDashboard({
      listings: [],
      listingsState: {
        data: { items: [], total: 30, page: 9, limit: 20, total_pages: 2 },
      },
    });

    expect(screen.queryByText("You haven't posted a listing yet.")).not.toBeInTheDocument();
    expect(screen.getByText("30 results")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
  });

  test("shows no page controls when everything fits on one", () => {
    renderDashboard({ listings: [makeListing()] });
    expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    expect(screen.queryByText(/Page 1 of/)).not.toBeInTheDocument();
  });

  test("a signed-out visitor is offered the login and neither query fires", () => {
    renderDashboard({ user: null });

    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    expect(useOrders).toHaveBeenCalledWith({ enabled: false });
    expect(useSearchListings).toHaveBeenCalledWith(
      { seller_id: undefined, page: 1, limit: 20, include_sold_out: true },
      { enabled: false },
    );
  });
});
