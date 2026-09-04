import { createContext } from "react";
import type { User } from "../api/types";

export interface AuthContextValue {
  // null means signed out AND not known yet, so branching on this alone
  // renders logged-out UI to a signed-in user for one round trip. Read
  // isLoading first, or be sure a parent already does.
  user: User | null;
  // True until the session-restore attempt on mount finishes, so consumers can
  // hold off rendering logged-out UI while a cookie session is being revived.
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  // Signup signs the user in as well - the backend returns a token.
  signup: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  // Exchanges the refresh cookie (set by the OAuth callback redirect) for a
  // session. Resolves true on success, false if no recoverable session.
  //
  // `force: true` is for the OAuth callback: it ignores any token already in
  // localStorage (which belongs to whoever was signed in before) and always
  // exchanges the cookie, so the session lands on the newly-authorised user
  // rather than a stale one. The default path keeps the cheap /auth/me fast
  // path on mount.
  restoreSession: (opts?: { force?: boolean }) => Promise<boolean>;
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const ACCESS_TOKEN_KEY = "access_token";
