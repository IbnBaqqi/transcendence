import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { ApiKey, CreatedApiKey } from "./types";

export function useApiKeys() {
  return useQuery({
    queryKey: keys.apiKeys.list(),
    queryFn: async () => (await api.get<ApiKey[]>("/me/api-keys")).data ?? [],
  });
}

export function useCreateApiKey() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) =>
      (await api.post<CreatedApiKey>("/me/api-keys", { name })).data,

    // Deliberately not written into the cache: the response carries the secret,
    // and the list must never hold it. Refetch instead, which returns the row
    // without `key`.
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.apiKeys.all });
    },
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();

  return useMutation({
    // 204, so there is nothing to read.
    mutationFn: async (id: string) => {
      await api.delete(apiPath`/me/api-keys/${id}`);
    },

    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.apiKeys.all });
    },
  });
}
