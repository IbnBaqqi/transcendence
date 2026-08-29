import { render, screen } from "@testing-library/react";

import { ListingCard } from "../components/objects/ListingCard";
import { makeListing } from "../test/factories";

describe("ListingCard", () => {
  test("shows the category name it is given, not the listing's slug", () => {
    render(
      <ListingCard listing={makeListing({ category: "mushrooms" })} categoryName="Mushrooms" />,
    );

    expect(screen.getByText("Mushrooms")).toBeInTheDocument();
    expect(screen.queryByText("mushrooms")).not.toBeInTheDocument();
  });
});
