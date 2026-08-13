import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { api } from "../api/client";
import type { User, AuthResponse } from "../api/types";

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void; // add this logic later
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const ACCESS_TOKEN_KEY = "accessToken"; // must match the key api/client.ts's interceptor reads

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem(ACCESS_TOKEN_KEY);
    if (stored) {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
    }
    setIsLoading(false);
  }, []);

  async function login(email: string, password: string) {
    const res = await api.post<AuthResponse>(
      "/login",
      { email, password },
      { withCredentials: true },
    );
    const { accessToken, user } = res.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    setUser(user);
  }

  function logout() {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    setUser(null);
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
