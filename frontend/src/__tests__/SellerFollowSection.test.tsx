import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { SellerFollowSection } from "../components/objects/SellerFollowSection";
import { useListing } from "../api/listings";
import { usePublicProfile } from "../api/profile";
import { useFollow, useFollowing, useUnfollow } from "../api/follows";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useModal } from "../providers/modalContext";
import { makeListing, makePublicProfile, BUYER_ID, SELLER_ID } from "../test/factories";
import type { User } from "../api/types";

vi.mock("../api/listings");
vi.mock("../api/profile");
vi.mock("../api/follows");
vi.mock("../providers/modalContext", () => ({ useModal: vi.fn() }));

const VIEWER: User = { id: BUYER_ID, username: "tester", email: "t@example.com", role: "user" };
const SELLER = makePublicProfile({ id: SELLER_ID, username: "mushroom_matti" });

beforeEach(() => {
  vi.mocked(useModal).mockReturnValue({ openModal: vi.fn() } as unknown as ReturnType<
    typeof useModal
  >);
  vi.mocked(useFollowing).mockReturnValue({
    data: [],
    isPending: false,
  } as unknown as ReturnType<typeof useFollowing>);
  const idle = { mutateAsync: vi.fn(), isPending: false };
  vi.mocked(useFollow).mockReturnValue(idle as unknown as ReturnType<typeof useFollow>);
  vi.mocked(useUnfollow).mockReturnValue(idle as unknown as ReturnType<typeof useUnfollow>);
});

function renderSection(listingId: string, user: User | null = VIEWER) {
  vi.mocked(useListing).mockReturnValue({
    data: makeListing({ seller_id: SELLER_ID }),
    isPending: false,
  } as unknown as ReturnType<typeof useListing>);
  vi.mocked(usePublicProfile).mockReturnValue({
    data: SELLER,
    isLoading: false,
  } as unknown as ReturnType<typeof usePublicProfile>);

  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={auth}>
        <MemoryRouter>
          <SellerFollowSection listingId={listingId} />
        </MemoryRouter>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("SellerFollowSection", () => {
  test("names the seller it offers to follow", () => {
    renderSection("l1");

    expect(screen.getByRole("link", { name: "mushroom_matti" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Follow" })).toBeInTheDocument();
  });

  // The listing carries only seller_id, so the name comes from a second query
  // keyed on it. Reading anything else here would show the wrong person.
  test("looks the seller up by the listing's seller_id", () => {
    renderSection("l1");
    expect(usePublicProfile).toHaveBeenLastCalledWith(SELLER_ID);
  });

  test("offers no follow button on your own listing", () => {
    renderSection("l1", { ...VIEWER, id: SELLER_ID });

    expect(screen.getByRole("link", { name: "mushroom_matti" })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  test("renders nothing without a listing id", () => {
    const { container } = renderSection("");
    expect(container).toBeEmptyDOMElement();
  });
});
