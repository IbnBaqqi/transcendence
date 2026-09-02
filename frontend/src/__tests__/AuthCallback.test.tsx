import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import AuthCallback from "../pages/AuthCallback";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";

// AuthCallback only needs restoreSession from the context; the provider itself
// would hit the network, so hand it a stub directly.
type RestoreMock = ReturnType<typeof restoreMock>;

function restoreMock() {
  return vi.fn<AuthContextValue["restoreSession"]>().mockResolvedValue(true);
}

function renderCallback(query: string, restore: RestoreMock) {
  const value: AuthContextValue = {
    user: null,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: restore,
  };

  render(
    <AuthContext.Provider value={value}>
      <MemoryRouter initialEntries={[`/auth/callback${query}`]}>
        <Routes>
          <Route path="/auth/callback" element={<AuthCallback />} />
          {/* probe: on success the page navigates away and this renders */}
          <Route path="/" element={<div>home-page</div>} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );

  return restore;
}

describe("AuthCallback error slugs", () => {
  test("maps access_denied to friendly cancel copy and a 'Sign-in cancelled' heading", () => {
    renderCallback("?error=access_denied", restoreMock());

    expect(screen.getByText("Sign-in cancelled")).toBeInTheDocument();
    expect(screen.getByText("Sign-in was cancelled. No changes were made.")).toBeInTheDocument();
  });

  test("email_in_use suggests the email+password path without promising a link flow", () => {
    renderCallback("?error=email_in_use", restoreMock());

    expect(screen.getByText("Sign-in didn't complete")).toBeInTheDocument();
    expect(
      screen.getByText(
        "An account with this email already exists. Sign in with your password instead.",
      ),
    ).toBeInTheDocument();
    // The old copy promised an unbuilt 'link from your profile' step - ensure it's gone.
    expect(screen.queryByText(/link your provider account/i)).not.toBeInTheDocument();
  });

  test("an unknown slug falls back to the server_error copy", () => {
    renderCallback("?error=not_a_real_slug", restoreMock());

    expect(
      screen.getByText("Something went wrong on our end. Please try again."),
    ).toBeInTheDocument();
  });
});

describe("AuthCallback success path", () => {
  test("forces a cookie exchange and navigates home when it succeeds", async () => {
    const restore = restoreMock();
    renderCallback("", restore);

    await waitFor(() => expect(restore).toHaveBeenCalledWith({ force: true }));
    await screen.findByText("home-page");
  });

  test("shows a generic failure when the exchange reports no session", async () => {
    const restore = vi.fn<AuthContextValue["restoreSession"]>().mockResolvedValue(false);
    renderCallback("", restore);

    expect(await screen.findByText("Sign-in didn't complete")).toBeInTheDocument();
    expect(restore).toHaveBeenCalledWith({ force: true });
  });
});

describe("AuthCallback error path", () => {
  test("does not attempt a session restore when an error slug is present", () => {
    const restore = restoreMock();
    renderCallback("?error=no_email", restore);

    expect(
      screen.getByText("Your provider account doesn't have a verified email."),
    ).toBeInTheDocument();
    expect(restore).not.toHaveBeenCalled();
  });
});
