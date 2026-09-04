import { render, screen } from "@testing-library/react";

import { SellerRating } from "../components/objects/SellerRating";

const show = (average: number, count: number) =>
  render(<SellerRating rating={{ average, count }} />);

// The whole point of the component. An unrated seller is {average: 0, count: 0},
// so reading the average alone brands everyone new as a zero-star seller.
test("says new seller rather than showing a zero", () => {
  show(0, 0);
  expect(screen.getByText("New seller")).toBeInTheDocument();
  expect(screen.queryByText(/0\.0/)).not.toBeInTheDocument();
});

// Five is the threshold, so four is new and five is rated. Testing either side
// of the boundary rather than a comfortable distance from it.
test("four reviews is still a new seller", () => {
  show(5, 4);
  expect(screen.getByText("New seller")).toBeInTheDocument();
});

test("five reviews earns a score", () => {
  show(4.5, 5);
  expect(screen.getByText("4.5 ★ (5)")).toBeInTheDocument();
  expect(screen.queryByText("New seller")).not.toBeInTheDocument();
});

// A seller can be rated badly and still be rated - the tag is about evidence,
// not about being good.
test("a genuinely bad rating is shown, not hidden behind the tag", () => {
  show(1.2, 40);
  expect(screen.getByText("1.2 ★ (40)")).toBeInTheDocument();
  expect(screen.queryByText("New seller")).not.toBeInTheDocument();
});

// The average is a float off the wire and 4.333333 is noise, not detail.
test("rounds the average to one decimal", () => {
  show(4.333333, 12);
  expect(screen.getByText("4.3 ★ (12)")).toBeInTheDocument();
});
