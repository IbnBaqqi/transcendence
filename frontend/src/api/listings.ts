import { useQuery } from "@tanstack/react-query";
import { api } from "./client";
import type { ApiResponse, Listing } from "./types";

// plain fetch function. uses axios client, returns typed data
async function fetchListings(): Promise<Listing[]> {
  const res = await api.get<ApiResponse<Listing[]>>("/listings");
  // res.data -> the envelope { success, data }
  // res.data.data -> the listings array (or null when empty), default to []
  // axios puts the HTTP body in res.data, and the body itself is { success, data },
  // so the array lives at res.data.data
  return res.data.data ?? [];
}

// the hook components will call, React Query wraps fetchListings with caching,
// loading/error states, dedup, etc.
export function useListings() {
  return useQuery({
    queryKey: ["listings"], // cache identity. same key = same shared cache entry
    queryFn: fetchListings, // how to fetch when the cache is empty/stale
  });
}
