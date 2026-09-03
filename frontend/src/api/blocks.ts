import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { BlockedUser } from "./types";

export function useBlocks(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: keys.me.blocks(),
    queryFn: async () => (await api.get<BlockedUser[]>("/me/blocks")).data ?? [],
    enabled: options.enabled ?? true,
  });
}

function useBlockMutation(method: "post" | "delete") {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (userId: string) => {
      await api[method](apiPath`/users/${userId}/block`);
      return userId;
    },
    onSuccess: (userId) => invalidateBlock(qc, userId),
  });
}

export const useBlock = () => useBlockMutation("post");
export const useUnblock = () => useBlockMutation("delete");

// A block hides presence in either direction and closes the thread (#146), so
// the profile and the inbox are both stale afterwards, not just the list.
function invalidateBlock(qc: QueryClient, userId: string) {
  return Promise.all([
    qc.invalidateQueries({ queryKey: keys.me.blocks() }),
    qc.invalidateQueries({ queryKey: keys.users.detail(userId) }),
    qc.invalidateQueries({ queryKey: keys.conversations.all }),
  ]);
}
