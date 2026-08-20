import { createContext } from "react";
import type { User } from "../api/types";

export interface AuthContextValue {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const ACCESS_TOKEN_KEY = "access_token";
