import { useState, type ReactNode } from "react";
import { api } from "../api/client";
import type { User, AuthResponse } from "../api/types";
import { AuthContext, ACCESS_TOKEN_KEY } from "./AuthContext";

export function AuthProvider({ children }: { children: ReactNode }) {
  // TODO: Replace with an async auth check (e.g. verify token via /auth/me)
  // once the backend auth flow is wired up. Until then, isLoading stays false
  // because the localStorage check is trivially fast.
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem(ACCESS_TOKEN_KEY);
    if (stored) localStorage.removeItem(ACCESS_TOKEN_KEY);
    return null;
  });

  async function login(email: string, password: string) {
    const res = await api.post<AuthResponse>(
      "/login",
      { email, password },
      { withCredentials: true },
    );
    const { access_token, user } = res.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, access_token);
    setUser(user);
  }

  function logout() {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    setUser(null);
  }

  return <AuthContext.Provider value={{ user, login, logout }}>{children}</AuthContext.Provider>;
}
