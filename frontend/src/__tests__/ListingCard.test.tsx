import { render, screen } from "@testing-library/react";

import { ListingCard } from "../components/objects/ListingCard";
import { makeListing } from "../test/factories";
import type { Listing } from "../api/types";

function renderCard(listing: Listing, showSeller?: boolean) {
  return render(<ListingCard listing={listing} categoryName="Mushrooms" showSeller={showSeller} />);
}

describe("ListingCard", () => {
  test("shows the category name it is given, not the listing's slug", () => {
    renderCard(makeListing({ category: "mushrooms" }));

    expect(screen.getByText("Mushrooms")).toBeInTheDocument();
    expect(screen.queryByText("mushrooms")).not.toBeInTheDocument();
  });

  test("names the seller and wears their picture", () => {
    const { container } = renderCard(
      makeListing({
        seller: { id: "s1", username: "mushroom_matti", avatar_url: "/uploads/matti.png" },
      }),
    );

    expect(screen.getByText("mushroom_matti")).toBeInTheDocument();
    expect(container.querySelector("img")).toHaveAttribute("src", "/uploads/matti.png");
  });

  // The initials fallback is the default avatar, so a seller who never
  // uploaded still gets a face rather than an empty circle.
  test("falls back to initials when the seller has no picture", () => {
    const { container } = renderCard(makeListing());

    expect(container.querySelector("img")).toBeNull();
    expect(screen.getByText("M")).toBeInTheDocument();
  });

  // seller is nullable because null means the lookup failed, not that nobody
  // posted it - the card degrades instead of rendering an empty row.
  test("renders no seller row when the API sent none", () => {
    renderCard(makeListing({ seller: null }));

    expect(screen.queryByText("mushroom_matti")).not.toBeInTheDocument();
  });

  test("a seller's own page suppresses the row that would name them on every card", () => {
    renderCard(makeListing(), false);

    expect(screen.queryByText("mushroom_matti")).not.toBeInTheDocument();
  });
});
