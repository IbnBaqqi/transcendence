import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { ListingCard } from "../components/objects/ListingCard";
import { makeListing } from "../test/factories";
import type { Listing } from "../api/types";

function renderCard(listing: Listing) {
  return render(
    <MemoryRouter>
      <ListingCard listing={listing} categoryName="Mushrooms" />
    </MemoryRouter>,
  );
}

describe("ListingCard", () => {
  test("shows the category name it is given, not the listing's slug", () => {
    renderCard(makeListing({ category: "mushrooms" }));

    expect(screen.getByText("Mushrooms")).toBeInTheDocument();
    expect(screen.queryByText("mushrooms")).not.toBeInTheDocument();
  });

  // Home is the only browse surface (#25 is still a stub), so without this the
  // only way to reach a listing is to paste its id into the URL bar.
  test("the whole card is a link to the listing", () => {
    const listing = makeListing();
    renderCard(listing);

    expect(screen.getByRole("link")).toHaveAttribute("href", `/listings/${listing.id}`);
  });

  // The card is an anchor now, so anything added inside it must not be one:
  // a nested anchor is invalid HTML, and the browser's recovery activates
  // both on a single click.
  test("contains exactly one link", () => {
    const { container } = renderCard(makeListing());
    expect(container.querySelectorAll("a")).toHaveLength(1);
  });
});
