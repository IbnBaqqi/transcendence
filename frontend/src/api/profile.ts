import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { ChangePasswordInput, OwnProfile, ProfileUpdateInput, PublicProfile } from "./types";

export function useOwnProfile() {
  return useQuery({
    queryKey: keys.me.profile(),
    queryFn: async () => (await api.get<OwnProfile>("/me/profile")).data,
  });
}

// GET /users/{id} - somebody else's profile.
// `id` is optional because it comes from a route param, which is
// string | undefined until React Router has matched the URL.
export function usePublicProfile(id: string | undefined) {
  return useQuery({
    // A key must be a valid string even while the query is disabled.
    queryKey: keys.users.detail(id ?? ""),
    queryFn: async () => {
      // Named so apiPath's signature can stay strict rather than accepting
      // undefined; `enabled` below is what stops this running without an id.
      const userId = id ?? "";
      return (await api.get<PublicProfile>(apiPath`/users/${userId}`)).data;
    },

    // Don't fire the request before the route param has been read.
    enabled: Boolean(id),

    // The default is retry: 1 (lib/queryClient.ts). A 404 here is a real
    // answer - no point asking twice.
    retry: false,
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: async (input: ChangePasswordInput) => {
      await api.post("/me/password", input, { withCredentials: true });
    },
  });
}

export function useUpdateOwnProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: ProfileUpdateInput) =>
      (await api.patch<OwnProfile>("/me/profile", input)).data,
    onSuccess: (updated) => {
      // Seed the cache with the server's answer so the change renders
      // immediately, then invalidate to catch anything a refetch would see.
      queryClient.setQueryData(keys.me.profile(), updated);
      queryClient.invalidateQueries({ queryKey: keys.me.profile() });
    },
  });
}
