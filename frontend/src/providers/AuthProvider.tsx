import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  getCurrentUser,
  login as loginApi,
  logout as logoutApi,
  refresh as refreshApi,
  signup as signupApi,
} from "../api/auth";
import { setOnSessionChange } from "../api/client";
import type { AuthResponse, User } from "../api/types";
import { AuthContext, ACCESS_TOKEN_KEY } from "./AuthContext";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const queryClient = useQueryClient();
  // True until the restore attempt below finishes. Starting optimistic means
  // consumers can wait out one round trip instead of flashing logged-out UI
  // on every page load.
  const [isLoading, setIsLoading] = useState(true);
  // Flips on unmount so a restore that resolves after the provider is gone
  // doesn't setState on an abandoned component. The mount guard used to live in
  // the effect closure; it has to reach inside restoreSession now, so it moves
  // to a ref.
  const mountedRef = useRef(true);

  const storeSession = useCallback(
    (res: AuthResponse) => {
      if (!mountedRef.current) return;
      localStorage.setItem(ACCESS_TOKEN_KEY, res.access_token);
      setUser(res.user);
      // Anything cached so far (e.g. a 401 on /me/profile from before this
      // login) was fetched signed-out or as somebody else - drop it so pages
      // already on screen, like Profile after its "Log In" button, refetch
      // under the new session instead of continuing to show that stale result.
      queryClient.clear();
    },
    [queryClient],
  );

  // Revives the current session if one exists: either answer /auth/me with the
  // live access token, or exchange the refresh cookie for a fresh one. Resolves
  // true when a user is now signed in.
  //
  // Mount: cheap /auth/me fast path when a token exists, refresh cookie
  // otherwise.
  // OAuth callback (force): the cookie is the authoritative new identity, so a
  // stale token must not win. Clear it first, then always exchange the cookie -
  // if that fails, the stale session is already gone and we stay signed out.
  const restoreSession = useCallback(
    async (opts?: { force?: boolean }): Promise<boolean> => {
      try {
        if (opts?.force) {
          // Drop whoever was signed in before the callback; the refresh cookie
          // is the new source of truth (a leftover token would /auth/me to the
          // wrong user). On failure there is no session to fall back to.
          localStorage.removeItem(ACCESS_TOKEN_KEY);
          storeSession(await refreshApi());
        } else if (localStorage.getItem(ACCESS_TOKEN_KEY)) {
          // Token still around: /auth/me answers identity without rotating a
          // perfectly good session. If it already expired, the interceptor
          // silently refreshes first and this call just succeeds.
          if (!mountedRef.current) return false;
          setUser(await getCurrentUser());
        } else {
          // No token, but a valid refresh cookie may still be sitting there.
          storeSession(await refreshApi());
        }
        return true;
      } catch {
        // No recoverable session; stay signed out.
        return false;
      }
    },
    [storeSession],
  );

  useEffect(() => {
    // Background refreshes (see the 401 interceptor) funnel through here so
    // React state and localStorage never disagree about who is signed in.
    setOnSessionChange((auth) => setUser(auth?.user ?? null));

    async function restoreOnMount() {
      await restoreSession();
      if (mountedRef.current) setIsLoading(false);
    }

    void restoreOnMount();
    return () => {
      mountedRef.current = false;
      setOnSessionChange(null);
    };
  }, [restoreSession]);

  async function login(email: string, password: string) {
    storeSession(await loginApi({ email, password }));
  }

  async function signup(username: string, email: string, password: string) {
    storeSession(await signupApi({ username, email, password }));
  }

  async function logout() {
    try {
      // Ends every session server-side and expires the refresh cookie.
      await logoutApi();
    } catch {
      // Local sign-out must survive a dead network or an expired token.
    }
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    setUser(null);
    // Every cached query (profile, orders, conversations, ...) was scoped to
    // the session that just ended - drop it all so the next signed-in user
    // (or a re-login as the same one) never renders someone else's stale data.
    queryClient.clear();
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, signup, logout, restoreSession }}>
      {children}
    </AuthContext.Provider>
  );
}
