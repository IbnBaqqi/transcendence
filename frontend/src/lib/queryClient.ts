import { QueryClient } from "@tanstack/react-query";

// the QueryClient holds the cache and default behaviour for every query in the app.
// you create it ONCE and share it through the provider (next file).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // how long data is treated as fresh (within this time window
      // reuses the same data, no refetch).
      // after, data is "stale" and is eligible for a background refresh
      staleTime: 30_000, // 30 seconds

      // retry failed request
      retry: 1,

      // React Query refetches by default when the browser tab regains focus
      // turned off to avoid surprise refetches during development
      // TODO: decide if this is changed later
      refetchOnWindowFocus: false,
    },
  },
});
