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
  // Whose answers the cache currently holds. A ref rather than reading `user`,
  // because the session-change callback below is registered once and would
  // otherwise close over the value from that first render. One effect keeps it
  // true whichever path set the user, so login and logout need no bookkeeping.
  const viewerIdRef = useRef<string | null>(null);

  useEffect(() => {
    viewerIdRef.current = user?.id ?? null;
  }, [user]);

  const storeSession = useCallback(
    (res: AuthResponse) => {
      if (!mountedRef.current) return;
      localStorage.setItem(ACCESS_TOKEN_KEY, res.access_token);
      setUser(res.user);
      // Anything cached so far (e.g. a 401 on /me/profile from before this
      // login) was fetched signed-out or as somebody else - drop it so pages
      // already on screen, like Profile after its "Log In" button, refetch
      // under the new session instead of continuing to show that stale result.
      //
      // Unconditional, unlike the identity guard on setOnSessionChange below.
      // Not an oversight: login, signup and the cookie exchange all arrive at
      // mount or after the old token is gone, so there is no warm cache to
      // protect and nothing an identity comparison would save.
      queryClient.clear();
    },
    [queryClient],
  );

  // Revives the current session if one exists: either answer /auth/me with the
  // live access token, or exchange the refresh cookie for a fresh one. Resolves
  // true when a user is now signed in.
  //
  // Mount: cheap /auth/me fast path when a token exists, and nothing at all
  // when one does not.
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
          //
          // The third path that sets a viewer, and the only one that never
          // clears the cache. It does not need to: restoreSession's callers are
          // restoreOnMount below, where nothing is cached yet, and
          // AuthCallback.tsx:42, which passes force and so takes the branch
          // above. Give it a caller that runs with a warm cache and it needs
          // the same identity guard setOnSessionChange has.
          if (!mountedRef.current) return false;
          setUser(await getCurrentUser());
        } else {
          // No token means no session to revive. It is only ever removed when
          // one ends - logout, or a refresh that already failed - so its
          // absence stands in for the HttpOnly refresh cookie, which scripts
          // cannot read. Asking anyway costs a 401 on every signed-out load,
          // and the browser logs that before our code can handle it.
          //
          // Below the force branch on purpose: the OAuth callback arrives with
          // a brand-new cookie and no token yet, and must not stop here.
          return false;
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
    mountedRef.current = true;
    // Background refreshes (see the 401 interceptor) funnel through here so
    // React state and localStorage never disagree about who is signed in.
    setOnSessionChange((auth) => {
      const next = auth?.user ?? null;
      // A refresh normally returns the same person, and clearing then would
      // throw away a warm cache on a routine token rotation. Only an identity
      // change means the cached answers were written for somebody else: keys
      // name the resource, never who asked, and GET /users/{id} carries
      // presence for a signed-in caller while omitting it for an anonymous one.
      //
      // Not folded into a setUser updater: those must be pure, and StrictMode
      // double-invokes them.
      if (viewerIdRef.current !== (next?.id ?? null)) queryClient.clear();
      setUser(next);
    });

    async function restoreOnMount() {
      await restoreSession();
      if (mountedRef.current) setIsLoading(false);
    }

    void restoreOnMount();
    return () => {
      mountedRef.current = false;
      setOnSessionChange(null);
    };
  }, [restoreSession, queryClient]);

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
