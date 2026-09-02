import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { ReserveListingSection } from "../components/forms/ReserveListingSection";
import { useListing } from "../api/listings";
import { useCreateOrder } from "../api/orders";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeListing, makeOrder, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Listing, User } from "../api/types";

vi.mock("../api/listings", () => ({ useListing: vi.fn() }));
vi.mock("../api/orders", () => ({ useCreateOrder: vi.fn() }));

const mutateAsync = vi.fn();

const VIEWER: User = {
  id: BUYER_ID,
  username: "tester",
  email: "t@example.com",
  role: "user",
};

beforeEach(() => {
  mutateAsync.mockReset().mockResolvedValue(makeOrder({ id: "o-new" }));
  vi.mocked(useCreateOrder).mockReturnValue({
    mutateAsync,
    isPending: false,
  } as unknown as ReturnType<typeof useCreateOrder>);
});

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

function renderSection(listing: Listing, user: User | null) {
  vi.mocked(useListing).mockReturnValue({
    data: listing,
    isPending: false,
  } as ReturnType<typeof useListing>);

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={authStub(user)}>
        <ModalProvider>
          <MemoryRouter initialEntries={["/listings/l1"]}>
            <Routes>
              <Route
                path="/listings/:id"
                element={<ReserveListingSection listingId={listing.id} />}
              />
              {/* probe: a successful reservation navigates here */}
              <Route path="/orders/:id" element={<div>order-page</div>} />
            </Routes>
          </MemoryRouter>
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("ReserveListingSection", () => {
  test("reserves the chosen quantity and sends the buyer to the new order", async () => {
    const user = userEvent.setup();
    const listing = makeListing({ quantity: 5 });
    renderSection(listing, VIEWER);

    await user.clear(screen.getByLabelText(/Quantity/));
    await user.type(screen.getByLabelText(/Quantity/), "3");
    await user.click(screen.getByRole("button", { name: "Request to buy" }));

    expect(mutateAsync).toHaveBeenCalledWith({ listing_id: listing.id, quantity: 3 });
    expect(await screen.findByText("order-page")).toBeInTheDocument();
  });

  test("refuses more than the seller has left", async () => {
    const user = userEvent.setup();
    renderSection(makeListing({ quantity: 2 }), VIEWER);

    await user.clear(screen.getByLabelText(/Quantity/));
    await user.type(screen.getByLabelText(/Quantity/), "9");

    const submit = screen.getByRole("button", { name: "Request to buy" });
    expect(submit).toBeDisabled();

    // Clicking it is the point: without this the assertion below passes on a
    // button that was never pressed.
    await user.click(submit);
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  test("a sold-out listing offers no form at all", () => {
    renderSection(makeListing({ quantity: 0 }), VIEWER);
    expect(screen.getByText("Sold out - nothing left to reserve.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Request to buy" })).not.toBeInTheDocument();
  });

  test("the seller can't reserve from themselves", () => {
    const seller: User = { ...VIEWER, id: SELLER_ID };
    renderSection(makeListing(), seller);
    expect(screen.getByText("This is your own listing.")).toBeInTheDocument();
  });

  test("a signed-out visitor is offered the login modal", () => {
    renderSection(makeListing(), null);
    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Request to buy" })).not.toBeInTheDocument();
  });

  // The stock race: the listing said 4, someone else took them first.
  test("a 409 shows the backend's stock message", async () => {
    mutateAsync.mockRejectedValue({ status: 409, message: "Not enough stock available" });
    const user = userEvent.setup();
    renderSection(makeListing({ quantity: 4 }), VIEWER);

    await user.click(screen.getByRole("button", { name: "Request to buy" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Not enough stock available");
  });
});
