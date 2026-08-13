import axios, { type AxiosError } from "axios";

// prod can override via env. dev falls back to "/api/v1"
const baseURL = import.meta.env.VITE_API_URL ?? "/api/v1";
export const api = axios.create({ baseURL });

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
    message: res?.data?.error ?? (res ? "Something went wrong" : "Could not reach the server"),
    details: res?.data?.details,
  };
}

// request: attach the auth token when present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("accessToken"); // placeholder store until #46
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// response: central place to handle errors, to be fleshed out later
api.interceptors.response.use(
  (res) => res,
  (error: AxiosError<ErrorBody>) => Promise.reject(toApiError(error)),
);

// catch clauses are typed `unknown`, so callers need this to narrow.
export function isApiError(e: unknown): e is ApiError {
  return typeof e === "object" && e !== null && "status" in e && "message" in e;
}
