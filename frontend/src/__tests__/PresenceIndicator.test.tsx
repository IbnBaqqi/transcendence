import { render, screen } from "@testing-library/react";

import { PresenceIndicator } from "../components/objects/PresenceIndicator";

test("says which state it is, rather than relying on the dot", () => {
  const { rerender } = render(<PresenceIndicator presence={{ is_online: true }} />);
  expect(screen.getByText("Online")).toBeInTheDocument();

  rerender(<PresenceIndicator presence={{ is_online: false }} />);
  expect(screen.getByText("Offline")).toBeInTheDocument();
});

// The dot carries no information a screen reader can use, and announcing it
// would read as a stray bullet before every status.
test("the dot is hidden from assistive tech", () => {
  const { container } = render(<PresenceIndicator presence={{ is_online: true }} />);
  expect(container.querySelector("[aria-hidden='true']")).toBeInTheDocument();
});
