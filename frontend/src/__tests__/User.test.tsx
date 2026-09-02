import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import User from "../pages/User";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { usePublicProfile } from "../api/profile";
import { useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { useFollow, useFollowers, useFollowing, useUnfollow } from "../api/follows";
import type { ApiError } from "../api/client";
import { makeListing, makePublicProfile } from "../test/factories";
// Aliased: "User" is already the page component above.
import type { Listing, Paginated, User as AuthUser } from "../api/types";

vi.mock("../api/profile");
vi.mock("../api/listings");
vi.mock("../api/categories");
vi.mock("../api/follows");

type ProfileQuery = ReturnType<typeof usePublicProfile>;
type FollowersQuery = ReturnType<typeof useFollowers>;
type ListingsQuery = ReturnType<typeof useSearchListings>;

const PROFILE = makePublicProfile();

// The page reads data.items now that the API does the filtering.
function page(items: Listing[]): Paginated<Listing> {
  return { items, total: items.length, page: 1, limit: 20, total_pages: 1 };
}

// The axios interceptor rejects with an ApiError object, not an Error, so
// React Query's error type has to be overridden here.
const NOT_FOUND: ApiError = { status: 404, message: "User not found" };

function authStub(user: AuthUser | null): AuthContextValue {
  return {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

// Both queries default to "loaded fine"; a test overrides only what it cares about.
function renderPage(
  opts: {
    profile?: Partial<ProfileQuery>;
    listings?: Partial<ListingsQuery>;
    currentUser?: AuthUser | null;
    // Defaults to the profile's own id; a test overrides it to visit the
    // same user under a different spelling of the id.
    urlId?: string;
  } = {},
) {
  vi.mocked(usePublicProfile).mockReturnValue({
    data: PROFILE,
    isLoading: false,
    error: null,
    ...opts.profile,
  } as ProfileQuery);

  vi.mocked(useSearchListings).mockReturnValue({
    data: page([]),
    isPending: false,
    isError: false,
    ...opts.listings,
  } as ListingsQuery);

  return render(
    // A real route, so useParams() reads the id out of the URL like in the app.
    <MemoryRouter initialEntries={[`/users/${opts.urlId ?? PROFILE.id}`]}>
      <AuthContext.Provider value={authStub(opts.currentUser ?? null)}>
        <ModalProvider>
          <Routes>
            <Route path="/users/:id" element={<User />} />
          </Routes>
        </ModalProvider>
      </AuthContext.Provider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.mocked(useLocalizedCategoryNames).mockReturnValue(
    (slug: string) => ({ mushrooms: "Mushrooms", berries: "Berries" })[slug] ?? slug,
  );

  vi.mocked(useFollowers).mockReturnValue({ data: [] } as unknown as FollowersQuery);
  vi.mocked(useFollowing).mockReturnValue({
    data: [],
    isPending: false,
  } as unknown as ReturnType<typeof useFollowing>);
  const idle = { mutateAsync: vi.fn(), isPending: false };
  vi.mocked(useFollow).mockReturnValue(idle as unknown as ReturnType<typeof useFollow>);
  vi.mocked(useUnfollow).mockReturnValue(idle as unknown as ReturnType<typeof useUnfollow>);
});

describe("User", () => {
  test("shows the fetched profile", () => {
    renderPage();
    expect(screen.getByText(PROFILE.username)).toBeInTheDocument();
    expect(screen.getByText("Forages the Nuuksio area.")).toBeInTheDocument();
    expect(screen.getByText("Helsinki")).toBeInTheDocument();
    expect(screen.getByText("Online")).toBeInTheDocument();
  });

  // The public API deliberately never sends these. If someone re-adds the
  // fields, this fails instead of shipping a leak-shaped UI.
  test("renders no email or phone number", () => {
    const { container } = renderPage();
    expect(container.querySelector('a[href^="mailto:"]')).toBeNull();
    expect(screen.queryByText("Telephone")).not.toBeInTheDocument();
  });

  // The guard is `{profile.presence && ...}`, which works because an object is
  // truthy even when is_online is false. Rewriting it as `presence?.is_online &&`
  // would silently drop this row for every offline user, with nothing else failing.
  test("shows Offline for a signed-in viewer when the user is offline", () => {
    renderPage({ profile: { data: makePublicProfile({ presence: { is_online: false } }) } });

    expect(screen.getByText("Offline")).toBeInTheDocument();
  });

  // The API omits presence for an anonymous caller, so this is what every
  // logged-out visitor receives. Dereferencing it unguarded throws, and the
  // boundary above the router would replace the whole app.
  test("renders an anonymous response, which carries no presence", () => {
    const anonymous = makePublicProfile();
    delete anonymous.presence;

    renderPage({ profile: { data: anonymous } });

    expect(screen.getByText(anonymous.username)).toBeInTheDocument();
    expect(screen.queryByText("Online")).not.toBeInTheDocument();
    expect(screen.queryByText("Offline")).not.toBeInTheDocument();
  });

  test("renders the 404 page when the user doesn't exist", () => {
    renderPage({
      profile: { data: undefined, error: NOT_FOUND as unknown as Error },
    });
    expect(screen.getByText(/page not found/i)).toBeInTheDocument();
  });

  // The filtering itself is the backend's now, and is tested there. What this
  // page still owns is asking the right question.
  test("asks the API for this seller's listings", () => {
    renderPage();
    expect(useSearchListings).toHaveBeenCalledWith(
      expect.objectContaining({ seller_id: PROFILE.id }),
    );
  });

  test("renders what the API sends back", () => {
    renderPage({ listings: { data: page([makeListing()]) } });
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
  });

  // Was a guard against comparing an upper-case URL id to a canonical one in
  // JavaScript. The comparison is Postgres's now, and uuid equality ignores
  // casing - so what is left to pin is that the page passes the id through
  // rather than trying to normalise it itself.
  test("passes an upper-case URL id straight through", () => {
    renderPage({ urlId: PROFILE.id.toUpperCase() });
    expect(useSearchListings).toHaveBeenCalledWith(
      expect.objectContaining({ seller_id: PROFILE.id.toUpperCase() }),
    );
  });

  test("offers a message button on someone else's profile", () => {
    renderPage({
      currentUser: {
        id: "99999999-9999-9999-9999-999999999999",
        username: "visitor",
        email: "v@example.com",
        role: "user",
      },
    });
    expect(screen.getByRole("button", { name: /message user/i })).toBeInTheDocument();
  });

  test("hides the message button on your own profile", () => {
    renderPage({
      currentUser: { id: PROFILE.id, username: "oscarroff", email: "o@example.com", role: "user" },
    });
    expect(screen.queryByRole("button", { name: /message user/i })).not.toBeInTheDocument();
  });

  // The follow button invalidates the key built from profile.id, so the count
  // has to be queried under that same string. Reading the URL's spelling here
  // would leave the number one behind after a follow, silently - query keys are
  // compared as strings, and nothing reports a miss.
  test("counts followers under the profile's id, not the URL's spelling", () => {
    // Last call, not any call: mock history is not cleared between tests here,
    // so toHaveBeenCalledWith would match an earlier test's render and pass
    // against a page that reads the URL id.
    const viewer: AuthUser = {
      id: "99999999-9999-9999-9999-999999999999",
      username: "visitor",
      email: "v@example.com",
      role: "user",
    };
    renderPage({ urlId: PROFILE.id.toUpperCase(), currentUser: viewer });
    expect(useFollowers).toHaveBeenLastCalledWith(PROFILE.id, viewer.id);
  });

  // GET /users/{id}/followers sits behind RequiredAuth, so asking without a
  // session is a guaranteed 401 that also burns a refresh on the interceptor.
  test("does not ask for followers when nobody is signed in", () => {
    renderPage({ currentUser: null });
    expect(useFollowers).toHaveBeenLastCalledWith(PROFILE.id, undefined);
  });

  test("counts the followers the API returned", () => {
    vi.mocked(useFollowers).mockReturnValue({
      data: [
        { id: "a", username: "one", presence: { is_online: true } },
        { id: "b", username: "two", presence: { is_online: false } },
      ],
    } as unknown as FollowersQuery);

    renderPage();
    expect(screen.getByText("2 followers")).toBeInTheDocument();
  });

  // n=1 is the only value that tells the plural forms apart, so a template
  // hardcoded to "followers" passes every other count.
  test("says follower, not followers, for exactly one", () => {
    vi.mocked(useFollowers).mockReturnValue({
      data: [{ id: "a", username: "one", presence: { is_online: true } }],
    } as unknown as FollowersQuery);

    renderPage();
    expect(screen.getByText("1 follower")).toBeInTheDocument();
  });

  // Undefined is "not answered yet", which is not the same as zero - claiming
  // zero followers before the request lands is a number we have not been told.
  test("shows no count until the followers request answers", () => {
    vi.mocked(useFollowers).mockReturnValue({ data: undefined } as unknown as FollowersQuery);

    renderPage();
    expect(screen.queryByText(/follower/)).not.toBeInTheDocument();
  });
});
