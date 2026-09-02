import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { AvatarResponse, OwnProfile, ProfileUpdateInput, PublicProfile } from "./types";

// Defaults to enabled: the profile page reads the 401 from this to decide it is
// signed out. Callers that already know there is no session pass false, so they
// do not spend a request and a refresh attempt learning it again.
export function useOwnProfile(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: keys.me.profile(),
    queryFn: async () => (await api.get<OwnProfile>("/me/profile")).data,
    enabled: options.enabled ?? true,
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

// userId so the viewer's own public profile is invalidated too: /me/profile and
// /users/{id} are separate cache entries holding the same avatar, and dropping
// this argument leaves one of them stale with nothing to report the miss.
export function useUploadAvatar(userId: string | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (file: File) => {
      const body = new FormData();
      body.append("avatar", file);
      // Content-Type is left to the browser: it carries a generated boundary,
      // and setting the header by hand produces a body the server cannot parse.
      return (await api.post<AvatarResponse>("/me/avatar", body)).data;
    },
    onSuccess: () => invalidateAvatar(queryClient, userId),
  });
}

export function useDeleteAvatar(userId: string | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await api.delete("/me/avatar");
    },
    onSuccess: () => invalidateAvatar(queryClient, userId),
  });
}

// Awaited by onSuccess, so isPending stays true until the new URL has actually
// arrived - otherwise the button frees up while the old avatar is still on
// screen.
function invalidateAvatar(queryClient: QueryClient, userId: string | undefined) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: keys.me.profile() }),
    userId
      ? queryClient.invalidateQueries({ queryKey: keys.users.detail(userId) })
      : Promise.resolve(),
  ]);
}
