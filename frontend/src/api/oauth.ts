import { useQuery } from "@tanstack/react-query";

import { api } from "./client";
import { keys } from "./queryKeys";

export async function fetchOAuthProviders(): Promise<string[]> {
  const res = await api.get<string[]>("/auth/providers");
  return res.data ?? [];
}

// Which providers the backend actually registered, which is only the ones whose
// credentials are set. staleTime Infinity because this is deployment config: it
// cannot change without a restart, and the same reasoning as useCategories.
export function useOAuthProviders() {
  return useQuery({
    queryKey: keys.oauth.providers(),
    queryFn: fetchOAuthProviders,
    staleTime: Infinity,
  });
}
