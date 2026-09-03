import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import Following from "../pages/Following";
import { useFollowing } from "../api/follows";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useModal } from "../providers/modalContext";
import { BUYER_ID, SELLER_ID } from "../test/factories";
import type { ChatUser, User } from "../api/types";

vi.mock("../api/follows", () => ({
  useFollowing: vi.fn(),
  useFollow: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
  useUnfollow: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
}));
vi.mock("../providers/modalContext", () => ({ useModal: vi.fn() }));

const VIEWER: User = { id: BUYER_ID, username: "tester", email: "t@example.com", role: "user" };

const ROWS: ChatUser[] = [
  {
    id: SELLER_ID,
    username: "mushroom_matti",
    avatar_url: "/uploads/matti.png",
    presence: { is_online: true },
  },
  {
    id: "44444444-4444-4444-4444-444444444444",
    username: "berry_bea",
    avatar_url: null,
    presence: { is_online: false },
  },
];

beforeEach(() => {
  vi.mocked(useModal).mockReturnValue({ openModal: vi.fn() } as unknown as ReturnType<
    typeof useModal
  >);
});

function authStub(user: User | null): AuthContextValue {
  return {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

function renderPage(
  query: { data?: ChatUser[]; isPending?: boolean; isError?: boolean },
  user: User | null,
) {
  vi.mocked(useFollowing).mockReturnValue({
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    ...query,
  } as unknown as ReturnType<typeof useFollowing>);

  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={authStub(user)}>
        <MemoryRouter>
          <Following />
        </MemoryRouter>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

describe("Following", () => {
  test("a signed-out visitor is asked to log in, not shown an error", () => {
    renderPage({ data: undefined }, null);

    expect(screen.getByRole("button", { name: "Log In" })).toBeInTheDocument();
    // Last call, not any call: mock history is not cleared between tests here,
    // so toHaveBeenCalledWith would start matching another test's render the
    // moment one is added above this one.
    expect(useFollowing).toHaveBeenLastCalledWith({ enabled: false });
  });

  // The screen has to demo both halves of the subject's bullet at once: who you
  // follow, and whether they are online.
  test("each row carries the username and that person's presence", () => {
    renderPage({ data: ROWS }, VIEWER);

    expect(screen.getByRole("link", { name: "mushroom_matti" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "berry_bea" })).toBeInTheDocument();
    expect(screen.getByText("Online")).toBeInTheDocument();
    expect(screen.getByText("Offline")).toBeInTheDocument();
  });

  test("an empty list explains how to fill it", () => {
    renderPage({ data: [] }, VIEWER);

    expect(screen.getByText("You're not following anyone yet.")).toBeInTheDocument();
    expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
  });
  // The fallback is the default avatar, so a row for someone who never
  // uploaded must still show something rather than an empty circle.
  test("wears each follow's picture, falling back to initials", () => {
    const { container } = renderPage({ data: ROWS }, VIEWER);

    const images = container.querySelectorAll("img");
    expect(images).toHaveLength(1);
    expect(images[0]).toHaveAttribute("src", "/uploads/matti.png");
    expect(screen.getByText("B")).toBeInTheDocument();
  });
});
