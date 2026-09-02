import axios, { type AxiosError } from "axios";

import type { AuthResponse } from "./types";
import { ACCESS_TOKEN_KEY } from "../providers/AuthContext";
import i18next from "../i18n";

// prod can override via env. dev falls back to "/api/v1". This is the base for
// both the axios instance below and the top-level OAuth navigation links
// (OAuthButtons), so they always agree no matter what VITE_API_URL is.
export const API_BASE_URL = import.meta.env.VITE_API_URL ?? "/api/v1";
export const api = axios.create({ baseURL: API_BASE_URL });

// Every failed request rejects with this, so callers handle one shape instead
// of digging through axios internals.
export interface ApiError {
  status: number; // 0 when the request never reached the server
  message: string;
  details?: Record<string, string>;
}

interface ErrorBody {
  error?: string;
  details?: Record<string, string>;
}

// Exported so it can be tested directly, rather than through axios internals.
export function toApiError(error: AxiosError<ErrorBody>): ApiError {
  const res = error.response;

  return {
    status: res?.status ?? 0,
    // The backend hasn't been localized, so a message it actually sent is
    // passed through verbatim; only the client-synthesized fallbacks are
    // translated here, in whatever language is active at request time.
    message:
      res?.data?.error ??
      (res ? i18next.t("common.somethingWentWrong") : i18next.t("common.couldNotReachServer")),
    details: res?.data?.details,
  };
}

// Marks a replayed request so a second 401 can't loop back into refresh.
declare module "axios" {
  export interface InternalAxiosRequestConfig {
    _retriedAfterRefresh?: boolean;
  }
}

// request: attach the auth token when present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem(ACCESS_TOKEN_KEY);
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// --- silent refresh ---------------------------------------------------------
// A 401 from anywhere except /auth/* means the access token expired or is
// missing. POST /auth/refresh once (the HttpOnly cookie rides along on its
// own), then replay the original request under the new token. If the refresh
// also fails, the session is gone for good: drop the token and let the 401
// propagate so the caller shows its login prompt.

// These either carry no token yet (login/signup), are the refresh itself, or
// succeed regardless of auth state (logout) - refreshing on their 401s would
// only burn a round trip.
const NO_REFRESH_PATTERN = /^\/auth\/(login|signup|logout|refresh)$/;

let inFlightRefresh: Promise<void> | null = null;

// AuthProvider registers this so a background refresh can keep React state
// (and therefore the header) in step; null means "signed out".
let onSessionChange: ((auth: AuthResponse | null) => void) | null = null;

export function setOnSessionChange(cb: ((auth: AuthResponse | null) => void) | null) {
  onSessionChange = cb;
}

async function refreshSession(): Promise<void> {
  try {
    const res = await api.post<AuthResponse>("/auth/refresh", undefined, {
      withCredentials: true,
    });
    localStorage.setItem(ACCESS_TOKEN_KEY, res.data.access_token);
    onSessionChange?.(res.data);
  } catch (error) {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    onSessionChange?.(null);
    throw error;
  } finally {
    // Only clear once settled, so requests arriving mid-refresh share it...
    inFlightRefresh = null;
  }
}

// response: central place to handle errors, to be fleshed out later
api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError<ErrorBody>) => {
    const config = error.config;

    if (
      error.response?.status !== 401 ||
      !config ||
      config._retriedAfterRefresh ||
      NO_REFRESH_PATTERN.test(config.url ?? "")
    ) {
      return Promise.reject(toApiError(error));
    }

    config._retriedAfterRefresh = true;

    inFlightRefresh ??= refreshSession();
    try {
      await inFlightRefresh;
    } catch {
      // Refresh failed and cleaned up after itself; report the original 401.
      return Promise.reject(toApiError(error));
    }

    return api.request(config);
  },
);

// Builds a request path with every interpolated value encoded.
//
//   apiPath`/users/${"../me/profile"}`  ->  "/users/..%2Fme%2Fprofile"
//
// A raw ${id} would let a value containing "/" or ".." change WHICH endpoint
// is called: URLs collapse "../" before the request is sent, so "/users/" plus
// "../me/profile" resolves to a different route entirely. React Router hands
// back decoded params, so the encoding has to be put back here.
export function apiPath(parts: TemplateStringsArray, ...values: (string | number)[]): string {
  return parts.reduce(
    (out, part, i) => out + part + (i < values.length ? encodeURIComponent(String(values[i])) : ""),
    "",
  );
}

// catch clauses are typed `unknown`, so callers need this to narrow.
export function isApiError(e: unknown): e is ApiError {
  return typeof e === "object" && e !== null && "status" in e && "message" in e;
}
