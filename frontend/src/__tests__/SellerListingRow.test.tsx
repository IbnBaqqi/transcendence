import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { SellerListingRow } from "../components/objects/SellerListingRow";
import { deriveListingStats } from "../lib/sellerStats";
import { makeListing, makeOrder } from "../test/factories";
import type { Order } from "../api/types";

function renderRow(quantity: number, orders: Order[] = []) {
  const listing = makeListing({ quantity });
  return render(
    <MemoryRouter>
      <SellerListingRow listing={listing} stats={deriveListingStats(listing, orders)} />
    </MemoryRouter>,
  );
}

describe("SellerListingRow", () => {
  test("shows what is left against what was posted, once something has sold", () => {
    renderRow(2, [makeOrder({ quantity: 2, status: "confirmed" })]);
    expect(screen.getByText(/2 of 4 kg left/)).toBeInTheDocument();
    expect(screen.getByText(/2 sold/)).toBeInTheDocument();
  });

  // "4 of 4" is noise, and there is no sold count to show yet.
  test("an untouched listing just says how much is left", () => {
    renderRow(4);
    expect(screen.getByText(/4 kg left/)).toBeInTheDocument();
    expect(screen.queryByText(/of 4/)).not.toBeInTheDocument();
    expect(screen.queryByText(/sold/)).not.toBeInTheDocument();
  });

  test("a sold-out listing says so instead of reporting zero", () => {
    renderRow(0, [makeOrder({ quantity: 4, status: "completed" })]);
    expect(screen.getByText(/Sold out/)).toBeInTheDocument();
    expect(screen.getByText(/4 sold/)).toBeInTheDocument();
  });

  test("links to the listing", () => {
    renderRow(4);
    expect(screen.getByRole("link")).toHaveAttribute("href", `/listings/${makeListing().id}`);
  });
});
