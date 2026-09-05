import { render, screen } from "@testing-library/react";

import { Modal } from "../components/modal/Modal";

// Without role="dialog" the panel is an anonymous div: assistive tech never
// announces that a dialog opened, and aria-modal has nothing to sit on. Found
// by driving Firefox for #246, where getByRole("dialog") matched nothing.
test("the panel is a dialog, and the backdrop is not", () => {
  render(
    <Modal onClose={() => {}}>
      <p>body</p>
    </Modal>,
  );

  const panel = screen.getByRole("dialog");
  expect(panel).toHaveAttribute("aria-modal", "true");
  expect(panel).toHaveTextContent("body");

  // The backdrop is the click-to-close target; naming it the dialog would put
  // the overlay into the accessibility tree as the thing being read.
  expect(panel.parentElement).not.toHaveAttribute("role", "dialog");
});
