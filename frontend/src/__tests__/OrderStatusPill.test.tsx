import { render } from "@testing-library/react";

import { OrderStatusPill } from "../components/objects/OrderStatusPill";
import type { OrderStatus } from "../api/types";

const STATUSES: OrderStatus[] = ["pending", "confirmed", "completed", "cancelled", "refunded"];

// Asserted through the rendered class rather than the style map, which stays
// unexported: react-refresh forbids a component file exporting anything else.
function classesFor(status: OrderStatus) {
  const { container } = render(<OrderStatusPill status={status} label="x" />);
  return container.firstElementChild?.className ?? "";
}

// Both past collisions came from editing one entry in isolation: refunded first
// matched cancelled, then matched pending. A pill indistinguishable from
// another status is one the user cannot read.
test("every order status renders a visually distinct pill", () => {
  const rendered = STATUSES.map(classesFor);
  expect(new Set(rendered).size).toBe(STATUSES.length);
});

// A missing key yields "undefined" in the class string rather than throwing.
test("no status renders an undefined class", () => {
  STATUSES.forEach((status) => {
    expect(classesFor(status)).not.toContain("undefined");
  });
});
