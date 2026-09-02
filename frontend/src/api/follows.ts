import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { ChatUser } from "./types";

export function useFollowing(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: keys.follows.following(),
    queryFn: async () => (await api.get<ChatUser[]>("/me/following")).data ?? [],
    enabled: options.enabled ?? true,
    // Online is "seen within the last 2 minutes", so these rows go stale on the
    // clock with no server event to invalidate on. Drop this and the dots keep
    // showing whatever was true when the page was opened.
    refetchInterval: 60_000,
  });
}

export function useFollowers(id: string | undefined) {
  return useQuery({
    queryKey: keys.follows.followers(id ?? ""),
    queryFn: async () => {
      const userId = id ?? "";
      return (await api.get<ChatUser[]>(apiPath`/users/${userId}/followers`)).data ?? [];
    },
    enabled: Boolean(id),
    retry: false,
  });
}

function useFollowMutation(method: "post" | "delete") {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (userId: string) => {
      await api[method](apiPath`/users/${userId}/follow`);
      return userId;
    },

    onSuccess: (userId) => {
      void qc.invalidateQueries({ queryKey: keys.follows.following() });
      void qc.invalidateQueries({ queryKey: keys.follows.followers(userId) });
    },
  });
}

export const useFollow = () => useFollowMutation("post");
export const useUnfollow = () => useFollowMutation("delete");
