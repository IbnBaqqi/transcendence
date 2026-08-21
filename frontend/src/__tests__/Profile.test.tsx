import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import Profile from "../pages/Profile";
import { ModalProvider } from "../providers/ModalProvider";
import { ModalRoot } from "../components/modal/ModalRoot";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useOwnProfile } from "../api/profile";
import type { OwnProfile } from "../api/types";

vi.mock("../api/profile", () => ({
  useOwnProfile: vi.fn(),
  useUpdateOwnProfile: vi.fn(),
}));

const mockedProfile = vi.mocked(useOwnProfile);

const PROFILE: OwnProfile = {
  id: "u1",
  username: "or99",
  email: "oscarrogers@example.com",
  firstname: "Oscar",
  lastname: "Rogers",
  bio: null,
  phone_number: null,
  date_of_birth: null,
  location: "Espoo",
};

// Profile itself only needs the modal system (delete-account, login prompt).
// The auth stub exists because the login modal it can open needs a context.
const AUTH_STUB: AuthContextValue = {
  user: null,
  isLoading: false,
  login: vi.fn().mockResolvedValue(undefined),
  signup: vi.fn(),
  logout: vi.fn(),
};

// Only the three fields the page actually reads; the real query result type
// is richer than any stub needs.
function renderPage(query: { data?: OwnProfile; isLoading?: boolean; error?: unknown }) {
  mockedProfile.mockReturnValue(query as ReturnType<typeof useOwnProfile>);
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={AUTH_STUB}>
        <ModalProvider>
          <Profile />
          <ModalRoot />
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

test("shows a loading note while the profile is being fetched", () => {
  renderPage({ data: undefined, isLoading: true, error: null });
  expect(screen.getByText("Loading…")).toBeInTheDocument();
});

test("greets a signed-out visitor with a way to log in", async () => {
  const user = userEvent.setup();
  renderPage({
    data: undefined,
    isLoading: false,
    error: { status: 401, message: "authentication required" },
  });
  expect(screen.getByText(/You're signed out/, { exact: false })).toBeInTheDocument();

  await user.click(screen.getByRole("button", { name: "Log In" }));
  expect(screen.getByRole("heading", { name: "Log in" })).toBeInTheDocument();
});

test("renders the signed-in identity from the backend", () => {
  renderPage({ data: PROFILE, isLoading: false, error: null });

  expect(screen.getByText("or99")).toBeInTheDocument();
  expect(screen.getByText("oscarrogers@example.com")).toBeInTheDocument();
  // Two real names beat the username fallback for the avatar.
  expect(screen.getByText("OR")).toBeInTheDocument();
});

test("other failures surface their message rather than spinning forever", () => {
  renderPage({
    data: undefined,
    isLoading: false,
    error: { status: 500, message: "Something went wrong" },
  });
  expect(screen.getByText("Something went wrong")).toBeInTheDocument();
});
