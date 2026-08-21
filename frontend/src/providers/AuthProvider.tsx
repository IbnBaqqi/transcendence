import { useEffect, useState, type ReactNode } from "react";
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
  // True until the restore attempt below finishes. Starting optimistic means
  // consumers can wait out one round trip instead of flashing logged-out UI
  // on every page load.
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Background refreshes (see the 401 interceptor) funnel through here so
    // React state and localStorage never disagree about who is signed in.
    setOnSessionChange((auth) => setUser(auth?.user ?? null));

    let cancelled = false;

    async function restoreSession() {
      try {
        if (localStorage.getItem(ACCESS_TOKEN_KEY)) {
          // Token still around: /auth/me answers identity without rotating a
          // perfectly good session. If it already expired, the interceptor
          // silently refreshes first and this call just succeeds.
          const me = await getCurrentUser();
          if (!cancelled) setUser(me);
        } else {
          // No token, but a valid refresh cookie may still be sitting there.
          const res = await refreshApi();
          if (!cancelled) setUser(res.user);
        }
      } catch {
        // No recoverable session; stay signed out.
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }

    void restoreSession();
    return () => {
      cancelled = true;
    };
  }, []);

  function storeSession(res: AuthResponse) {
    localStorage.setItem(ACCESS_TOKEN_KEY, res.access_token);
    setUser(res.user);
  }

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
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, signup, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
