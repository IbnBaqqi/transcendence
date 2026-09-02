import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import User from "../pages/User";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { usePublicProfile } from "../api/profile";
import { useListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import type { ApiError } from "../api/client";
import { makeListing, makePublicProfile } from "../test/factories";
// Aliased: "User" is already the page component above.
import type { User as AuthUser } from "../api/types";

vi.mock("../api/profile");
vi.mock("../api/listings");
vi.mock("../api/categories");

type ProfileQuery = ReturnType<typeof usePublicProfile>;
type ListingsQuery = ReturnType<typeof useListings>;

const PROFILE = makePublicProfile();

// makeListing()'s default seller_id is PROFILE.id, so this is the odd one out.
const OTHER_SELLERS_LISTING = makeListing({
  id: "01a02305-b81d-764a-a738-d8c0642639de",
  seller_id: "99999999-9999-9999-9999-999999999999",
  title: "Wild Blueberries",
});

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

  vi.mocked(useListings).mockReturnValue({
    data: [],
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

  test("shows only this user's listings", () => {
    renderPage({ listings: { data: [makeListing(), OTHER_SELLERS_LISTING] } });
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
    expect(screen.queryByText("Wild Blueberries")).not.toBeInTheDocument();
  });

  // Regression guard: the API accepts an upper-case UUID and answers with the
  // canonical lower-case one, which is what listings carry. Filtering on the
  // URL param instead of profile.id renders the profile with no listings.
  test("matches listings against the canonical id, not the URL casing", () => {
    renderPage({
      urlId: PROFILE.id.toUpperCase(),
      listings: { data: [makeListing()] },
    });
    expect(screen.getByText("Golden Chanterelles")).toBeInTheDocument();
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
});
