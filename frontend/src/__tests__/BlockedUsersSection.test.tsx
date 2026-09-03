import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { BlockedUsersSection } from "../components/forms/BlockedUsersSection";
import { useBlocks, useUnblock } from "../api/blocks";
import { SELLER_ID } from "../test/factories";
import type { BlockedUser } from "../api/types";

vi.mock("../api/blocks", () => ({ useBlocks: vi.fn(), useUnblock: vi.fn() }));

const unblockCall = vi.fn();

const BLOCKED: BlockedUser = {
  id: SELLER_ID,
  username: "mushroom_matti",
  blocked_at: "1970-01-01T00:00:00Z",
};

beforeEach(() => {
  unblockCall.mockReset().mockResolvedValue(SELLER_ID);
  vi.mocked(useUnblock).mockReturnValue({
    mutateAsync: unblockCall,
    isPending: false,
  } as unknown as ReturnType<typeof useUnblock>);
});

function renderSection(query: { data?: BlockedUser[]; isPending?: boolean; isError?: boolean }) {
  vi.mocked(useBlocks).mockReturnValue({
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    ...query,
  } as unknown as ReturnType<typeof useBlocks>);

  return render(
    <MemoryRouter>
      <BlockedUsersSection />
    </MemoryRouter>,
  );
}

describe("BlockedUsersSection", () => {
  test("names each blocked person and links to them", () => {
    renderSection({ data: [BLOCKED] });

    expect(screen.getByRole("link", { name: "mushroom_matti" })).toHaveAttribute(
      "href",
      `/users/${SELLER_ID}`,
    );
  });

  test("unblocking asks the API for that person", async () => {
    const user = userEvent.setup();
    renderSection({ data: [BLOCKED] });

    await user.click(screen.getByRole("button", { name: "Unblock" }));

    expect(unblockCall).toHaveBeenCalledWith(SELLER_ID);
  });

  // An empty list is the normal case, so it has to read as "nothing to see"
  // rather than as a section that failed to load.
  test("an empty list says so rather than rendering nothing", () => {
    renderSection({ data: [] });

    expect(screen.getByText("You haven't blocked anyone.")).toBeInTheDocument();
    expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
  });
});
