import { useQuery } from "@tanstack/react-query";
import { api } from "./client";
import type { Listing } from "./types";

// GET /api/v1/listings -> a base JSON array of listings
//
// no response envelope: success vs failure is signalled by the HTTP status
// code. axios throws on any non-2xx, which React Query turns into isError,
// so the filaure path needs no special handling here
//
// NOTE: this targets the ListingResponse DTO from feature/listing-response-dto,
// which isnt merged yet. until it lands the backend still serialises raw sqlc
// structs (PascalCase keys, null-wrappers), so the feed will not render.

async function fetchListings(): Promise<Listing[]> {
  const res = await api.get<Listing[]>("/listings");
  // res.data IS the array here (axios puts the HTTP body on .data).
  // ?? [] is insurance in case an empty result ever arrives as null.
  return res.data ?? [];
}

// the hook components will call, React Query wraps fetchListings with caching,
// loading/error states, dedup, etc.
export function useListings() {
  return useQuery({
    queryKey: ["listings"], // cache identity. same key = same shared cache entry
    queryFn: fetchListings, // how to fetch when the cache is empty/stale
  });
}
