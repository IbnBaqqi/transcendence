import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { BlockButton } from "../components/objects/BlockButton";
import { useBlock, useBlocks, useUnblock } from "../api/blocks";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { BUYER_ID, SELLER_ID } from "../test/factories";
import type { BlockedUser, User } from "../api/types";

vi.mock("../api/blocks", () => ({
  useBlocks: vi.fn(),
  useBlock: vi.fn(),
  useUnblock: vi.fn(),
}));

const blockCall = vi.fn();
const unblockCall = vi.fn();

const VIEWER: User = {
  id: BUYER_ID,
  username: "tester",
  email: "t@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

const BLOCKED: BlockedUser = {
  id: SELLER_ID,
  username: "seller",
  blocked_at: "1970-01-01T00:00:00Z",
};

beforeEach(() => {
  blockCall.mockReset().mockResolvedValue(SELLER_ID);
  unblockCall.mockReset().mockResolvedValue(SELLER_ID);
  vi.mocked(useBlock).mockReturnValue({
    mutateAsync: blockCall,
    isPending: false,
  } as unknown as ReturnType<typeof useBlock>);
  vi.mocked(useUnblock).mockReturnValue({
    mutateAsync: unblockCall,
    isPending: false,
  } as unknown as ReturnType<typeof useUnblock>);
});

function renderButton(
  user: User | null,
  blocks: BlockedUser[] = [],
  authLoading = false,
  targetId = SELLER_ID,
) {
  vi.mocked(useBlocks).mockReturnValue({
    data: blocks,
    isPending: false,
  } as unknown as ReturnType<typeof useBlocks>);

  const auth: AuthContextValue = {
    user,
    isLoading: authLoading,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <AuthContext.Provider value={auth}>
      <BlockButton userId={targetId} />
    </AuthContext.Provider>,
  );
}

describe("BlockButton", () => {
  test("offers nothing on your own profile", () => {
    const { container } = renderButton(VIEWER, [], false, VIEWER.id);
    expect(container).toBeEmptyDOMElement();
  });

  // Blocking needs a session, so there is nothing to offer a visitor - and
  // AuthProvider reports null until the restore finishes, so the same branch
  // must cover both rather than flashing a button that would 401.
  test("offers nothing while signed out or still restoring", () => {
    expect(renderButton(null).container).toBeEmptyDOMElement();
    expect(renderButton(null, [], true).container).toBeEmptyDOMElement();
  });

  test("blocks someone absent from the list", async () => {
    const user = userEvent.setup();
    renderButton(VIEWER, []);

    await user.click(screen.getByRole("button", { name: "Block" }));

    expect(blockCall).toHaveBeenCalledWith(SELLER_ID);
    expect(unblockCall).not.toHaveBeenCalled();
  });

  // The label comes from the cached list, so getting this backwards blocks on
  // a click meant to unblock.
  test("unblocks someone already on the list", async () => {
    const user = userEvent.setup();
    renderButton(VIEWER, [BLOCKED]);

    await user.click(screen.getByRole("button", { name: "Unblock" }));

    expect(unblockCall).toHaveBeenCalledWith(SELLER_ID);
    expect(blockCall).not.toHaveBeenCalled();
  });

  test("surfaces the backend's refusal", async () => {
    blockCall.mockRejectedValue({ status: 400, message: "You cannot block yourself" });
    const user = userEvent.setup();
    renderButton(VIEWER, []);

    await user.click(screen.getByRole("button", { name: "Block" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("You cannot block yourself");
  });
});
