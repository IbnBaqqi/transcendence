import { createContext } from "react";
import type { User } from "../api/types";

export interface AuthContextValue {
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
  restoreSession: () => Promise<boolean>;
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const ACCESS_TOKEN_KEY = "access_token";
