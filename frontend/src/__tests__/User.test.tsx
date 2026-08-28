import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import User from "../pages/User";
import { ModalProvider } from "../providers/ModalProvider";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { usePublicProfile } from "../api/profile";
import { useListings } from "../api/listings";
import type { ApiError } from "../api/client";
import { makeListing, makePublicProfile } from "../test/factories";
// Aliased: "User" is already the page component above.
import type { User as AuthUser } from "../api/types";

vi.mock("../api/profile");
vi.mock("../api/listings");

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
  return { user, isLoading: false, login: vi.fn(), signup: vi.fn(), logout: vi.fn() };
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
