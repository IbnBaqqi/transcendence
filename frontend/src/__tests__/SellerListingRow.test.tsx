import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { SellerListingRow } from "../components/objects/SellerListingRow";
import { useDeleteListing } from "../api/listings";
import { deriveListingStats } from "../lib/sellerStats";
import { makeListing, makeOrder } from "../test/factories";
import type { Order, OrderStatus } from "../api/types";

// The row owns the delete action now, so every render needs the mutation.
vi.mock("../api/listings", () => ({ useDeleteListing: vi.fn() }));

const deleteListing = vi.fn();

beforeEach(() => {
  deleteListing.mockReset().mockResolvedValue(undefined);
  vi.mocked(useDeleteListing).mockReturnValue({
    mutateAsync: deleteListing,
    isPending: false,
  } as unknown as ReturnType<typeof useDeleteListing>);
});

function renderRow(quantity: number, orders: Order[] = []) {
  const listing = makeListing({ quantity });
  return render(
    <MemoryRouter>
      <SellerListingRow
        listing={listing}
        stats={deriveListingStats(listing, orders)}
        orders={orders}
      />
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

  // The card used to be one <Link> around everything, which cannot hold a
  // button - so the link moved to the title when the delete action arrived.
  test("links to the listing, without wrapping the controls in it", () => {
    renderRow(4);
    const title = screen.getByRole("link");
    expect(title).toHaveAttribute("href", `/listings/${makeListing().id}`);
    expect(title.querySelector("button")).toBeNull();
  });
});

describe("deleting", () => {
  // The rule the backend enforces, mirrored so the seller reads it before
  // clicking rather than after a 409. Both sides matter: blocking on a finished
  // sale is what made a listing permanent once anything had ever sold.
  test.each([
    ["pending", false],
    ["confirmed", false],
    ["completed", true],
    ["cancelled", true],
    ["refunded", true],
  ] as [OrderStatus, boolean][])("a %s order leaves delete available: %s", (status, available) => {
    renderRow(3, [makeOrder({ status })]);

    if (available) {
      expect(screen.getByRole("button", { name: "Delete listing" })).toBeInTheDocument();
    } else {
      expect(screen.queryByRole("button", { name: "Delete listing" })).not.toBeInTheDocument();
      expect(screen.getByText(/A sale is in progress/)).toBeInTheDocument();
    }
  });

  // Irreversible, and it takes the photos with it, so the first click only asks.
  test("takes two clicks, and says what goes", async () => {
    const user = userEvent.setup();
    renderRow(3);

    await user.click(screen.getByRole("button", { name: "Delete listing" }));
    expect(deleteListing).not.toHaveBeenCalled();
    expect(screen.getByText(/Completed sales keep their records/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Delete permanently" }));
    expect(deleteListing).toHaveBeenCalledWith(makeListing().id);
  });

  // The button the keyboard was standing on is replaced by two others. Without
  // the handoff focus lands on the body, and the confirm step can only be
  // reached by tabbing in again from the top of the page.
  test("moves focus to the confirm button when the step changes", async () => {
    const user = userEvent.setup();
    renderRow(3);

    await user.click(screen.getByRole("button", { name: "Delete listing" }));

    expect(screen.getByRole("button", { name: "Delete permanently" })).toHaveFocus();
  });

  // A sale can start between the render and the click, so the client's guess
  // can be stale - the server's answer is the one the seller needs to see.
  test("shows the server's reason when it refuses", async () => {
    deleteListing.mockRejectedValue({
      status: 409,
      message: "This listing has a sale in progress. Finish or cancel it before deleting.",
    });
    const user = userEvent.setup();
    renderRow(3);

    await user.click(screen.getByRole("button", { name: "Delete listing" }));
    await user.click(screen.getByRole("button", { name: "Delete permanently" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("sale in progress");
  });
});
