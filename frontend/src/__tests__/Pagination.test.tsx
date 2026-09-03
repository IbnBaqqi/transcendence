import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Pagination } from "../components/objects/Pagination";

function renderPagination(page: number, totalPages: number, total = 47) {
  const onPageChange = vi.fn();
  render(
    <Pagination page={page} totalPages={totalPages} total={total} onPageChange={onPageChange} />,
  );
  return { onPageChange, user: userEvent.setup() };
}

describe("Pagination", () => {
  test("the first page cannot go back, and next steps forward", async () => {
    const { onPageChange, user } = renderPagination(1, 3);

    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  test("the last page cannot go forward", () => {
    renderPagination(3, 3);

    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
  });

  test("a single page keeps the count but drops the controls", () => {
    renderPagination(1, 1, 3);

    expect(screen.getByText("3 results")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
