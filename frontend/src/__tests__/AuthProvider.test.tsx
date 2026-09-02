import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { StrictMode } from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthProvider } from "../providers/AuthProvider";
import { useAuth } from "../hooks/useAuth";
import * as authApi from "../api/auth";
import { ACCESS_TOKEN_KEY } from "../providers/AuthContext";

vi.mock("../api/auth");
const mockedAuthApi = vi.mocked(authApi);

const session = {
  access_token: "tok",
  user: { id: "u1", username: "forager", email: "f@example.com", role: "USER" as const },
};

// Stands in for a page's own query (e.g. Profile's useOwnProfile) that was
// already mounted - and fetched, possibly erroring if signed out - before the
// session changed. Counting calls (rather than reading the cache directly)
// sidesteps the race between queryClient.clear() and the automatic refetch
// it triggers for any query an observer like this is still watching.
const probeFetcher = vi.fn().mockResolvedValue("fresh");

function ProbeQuery() {
  const { status } = useQuery({
    queryKey: ["probe"],
    queryFn: probeFetcher,
    retry: false,
  });
  return <span>probe:{status}</span>;
}

function Consumer() {
  const { user, login, signup, logout, restoreSession } = useAuth();
  return (
    <>
      <span>{user ? user.username : "signed-out"}</span>
      <button onClick={() => void login("f@example.com", "secret12")}>Log In</button>
      <button onClick={() => void signup("forager", "f@example.com", "secret12")}>Register</button>
      <button onClick={() => void logout()}>Log Out</button>
      <button onClick={() => void restoreSession({ force: true })}>Force Restore</button>
      <ProbeQuery />
    </>
  );
}

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Consumer />
      </AuthProvider>
    </QueryClientProvider>,
  );
  return { queryClient };
}

// Dev renders the app under StrictMode, which mounts -> unmounts -> remounts the
// provider on first render. A bug where the mountedRef guard is never re-armed
// on remount makes every later storeSession a silent no-op (login/signup do
// nothing); this harness exists so that regression is visible.
function renderAppStrictMode() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <Consumer />
        </AuthProvider>
      </QueryClientProvider>
    </StrictMode>,
  );
  return { queryClient };
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  // No token yet: restoreSession falls through to a refresh attempt, which
  // has nothing to recover on a clean slate.
  mockedAuthApi.refresh.mockRejectedValue({ status: 401, message: "no session" });
});

test("a successful login drops every cached query so mounted pages refetch", async () => {
  mockedAuthApi.login.mockResolvedValue(session);
  const user = userEvent.setup();
  renderApp();

  await screen.findByText("probe:success");
  expect(probeFetcher).toHaveBeenCalledTimes(1);

  await user.click(screen.getByRole("button", { name: "Log In" }));

  await screen.findByText("forager");
  // The clear forces the still-mounted probe to refetch under the new session.
  await waitFor(() => expect(probeFetcher).toHaveBeenCalledTimes(2));
  expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("tok");
});

test("login still stores the session under StrictMode's mount-unmount-remount", async () => {
  mockedAuthApi.login.mockResolvedValue(session);
  const user = userEvent.setup();
  renderAppStrictMode();

  // StrictMode re-invokes the effect; wait for the (re)mount to settle.
  await screen.findByText("probe:success");

  await user.click(screen.getByRole("button", { name: "Log In" }));

  // If mountedRef isn't re-armed on remount, storeSession bails silently and
  // neither the user nor the token appear.
  await screen.findByText("forager");
  expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("tok");
});

test("a successful registration drops every cached query so mounted pages refetch", async () => {
  mockedAuthApi.signup.mockResolvedValue(session);
  const user = userEvent.setup();
  renderApp();

  await screen.findByText("probe:success");
  expect(probeFetcher).toHaveBeenCalledTimes(1);

  await user.click(screen.getByRole("button", { name: "Register" }));

  await screen.findByText("forager");
  await waitFor(() => expect(probeFetcher).toHaveBeenCalledTimes(2));
});

test("logging out also drops every cached query", async () => {
  mockedAuthApi.login.mockResolvedValue(session);
  mockedAuthApi.logout.mockResolvedValue(undefined);
  const user = userEvent.setup();
  renderApp();

  await user.click(screen.getByRole("button", { name: "Log In" }));
  await screen.findByText("forager");
  await waitFor(() => expect(probeFetcher).toHaveBeenCalledTimes(2));

  await user.click(screen.getByRole("button", { name: "Log Out" }));

  await screen.findByText("signed-out");
  // A query fetched while signed in gets cleared on the way out too.
  await waitFor(() => expect(probeFetcher).toHaveBeenCalledTimes(3));
  expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
});

test("a forced restore exchanges the cookie even when a stale token exists", async () => {
  // Old session's token is already in storage - the OAuth callback must not
  // trust it (it belongs to whoever was signed in before the redirect).
  localStorage.setItem(ACCESS_TOKEN_KEY, "stale");
  mockedAuthApi.refresh.mockResolvedValue(session);
  const user = userEvent.setup();
  renderApp();
  // The mount restore above already consulted /auth/me (it saw the stale token).
  // Clear that so we can assert the forced restore itself never does.
  mockedAuthApi.getCurrentUser.mockClear();

  await user.click(screen.getByRole("button", { name: "Force Restore" }));

  await screen.findByText("forager");
  // The stale token is replaced by a real cookie exchange, and the forced
  // restore never asks /auth/me (which would have answered "stale" and left the
  // cookie session unused until a silent refresh flipped the identity).
  await waitFor(() => expect(mockedAuthApi.refresh).toHaveBeenCalled());
  expect(mockedAuthApi.getCurrentUser).not.toHaveBeenCalled();
  expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("tok");
});

test("a forced restore that fails drops the stale token and stays signed out", async () => {
  localStorage.setItem(ACCESS_TOKEN_KEY, "stale");
  mockedAuthApi.refresh.mockRejectedValue({ status: 401, message: "no session" });
  const user = userEvent.setup();
  renderApp();

  await user.click(screen.getByRole("button", { name: "Force Restore" }));

  await screen.findByText("signed-out");
  // No fallback to the stale token: the callback cookie is authoritative and
  // failed, so the old identity must not linger.
  expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
});
