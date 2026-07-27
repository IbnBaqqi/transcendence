import axios from "axios";

// prod can override via env. dev falls back to "/api/v1"
const baseURL = import.meta.env.VITE_API_URL ?? "/api/v1";
export const api = axios.create({ baseURL });

// request: attach the auth token when present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("accessToken"); // placeholder store for now
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// response: central place to handle errors
api.interceptors.response.use(
  (res) => res,
  (error) => Promise.reject(error),
);
