import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { FollowButton } from "../components/objects/FollowButton";
import { useFollow, useFollowing, useUnfollow } from "../api/follows";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useModal } from "../providers/modalContext";
import { BUYER_ID, SELLER_ID } from "../test/factories";
import type { ChatUser, User } from "../api/types";

vi.mock("../api/follows", () => ({
  useFollowing: vi.fn(),
  useFollow: vi.fn(),
  useUnfollow: vi.fn(),
}));
vi.mock("../providers/modalContext", () => ({ useModal: vi.fn() }));

const followCall = vi.fn();
const unfollowCall = vi.fn();
const openModal = vi.fn();

type Mutation = ReturnType<typeof useFollow>;
const stub = (mutateAsync: unknown, isPending = false) =>
  ({ mutateAsync, isPending }) as unknown as Mutation;

const VIEWER: User = {
  id: BUYER_ID,
  username: "tester",
  email: "t@example.com",
  role: "user",
  has_password: true,
  providers: [],
};

const SELLER_ROW: ChatUser = {
  id: SELLER_ID,
  username: "seller",
  avatar_url: null,
  presence: { is_online: true },
};

beforeEach(() => {
  followCall.mockReset().mockResolvedValue(SELLER_ID);
  unfollowCall.mockReset().mockResolvedValue(SELLER_ID);
  openModal.mockReset();
  vi.mocked(useFollow).mockReturnValue(stub(followCall));
  vi.mocked(useUnfollow).mockReturnValue(stub(unfollowCall));
  vi.mocked(useModal).mockReturnValue({ openModal } as unknown as ReturnType<typeof useModal>);
});

function authStub(user: User | null, isLoading = false): AuthContextValue {
  return {
    user,
    isLoading,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

function renderButton(
  user: User | null,
  following: { data?: ChatUser[]; isPending?: boolean } = { data: [] },
  authLoading = false,
) {
  vi.mocked(useFollowing).mockReturnValue({
    isPending: false,
    ...following,
  } as ReturnType<typeof useFollowing>);

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={authStub(user, authLoading)}>
        <FollowButton userId={SELLER_ID} />
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("FollowButton", () => {
  test("offers nothing on your own profile", () => {
    const { container } = renderButton({ ...VIEWER, id: SELLER_ID });
    expect(container).toBeEmptyDOMElement();
  });

  // The button is real rather than a sentence, so the signed-out path has to
  // end at the login modal and NOT at a request that would 401.
  test("a signed-out visitor is sent to the login modal, not to the API", async () => {
    const user = userEvent.setup();
    renderButton(null);

    await user.click(screen.getByRole("button", { name: "Follow" }));

    expect(openModal).toHaveBeenCalledWith("login");
    expect(followCall).not.toHaveBeenCalled();
  });

  // AuthProvider reports user as null for the first render while it restores
  // the session, so a signed-in visitor would be offered the login modal on
  // every page load - and get it if they clicked before the restore landed.
  test("waits out the session restore instead of offering the login modal", async () => {
    const user = userEvent.setup();
    renderButton(null, { data: [] }, true);

    const button = screen.getByRole("button", { name: "Follow" });
    expect(button).toBeDisabled();

    await user.click(button);
    expect(openModal).not.toHaveBeenCalled();
  });

  test("reads Follow when the list does not contain them, and follows", async () => {
    const user = userEvent.setup();
    renderButton(VIEWER, { data: [] });

    await user.click(screen.getByRole("button", { name: "Follow" }));

    expect(followCall).toHaveBeenCalledWith(SELLER_ID);
    expect(unfollowCall).not.toHaveBeenCalled();
  });

  // The label is derived from the cached list, so a row for this user is the
  // only thing that flips it. Getting this backwards unfollows on a click
  // meant to follow.
  test("reads Unfollow when the list contains them, and unfollows", async () => {
    const user = userEvent.setup();
    renderButton(VIEWER, { data: [SELLER_ROW] });

    await user.click(screen.getByRole("button", { name: "Unfollow" }));

    expect(unfollowCall).toHaveBeenCalledWith(SELLER_ID);
    expect(followCall).not.toHaveBeenCalled();
  });

  // Until the list arrives we do not know which action this is, and a label
  // that guesses is a label that sometimes lies.
  test("stays disabled until the following list has arrived", () => {
    renderButton(VIEWER, { data: undefined, isPending: true });
    expect(screen.getByRole("button", { name: "Follow" })).toBeDisabled();
  });

  test("shows the backend's message when the request fails", async () => {
    followCall.mockRejectedValue({ status: 404, message: "User not found" });
    const user = userEvent.setup();
    renderButton(VIEWER, { data: [] });

    await user.click(screen.getByRole("button", { name: "Follow" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("User not found");
  });
});
