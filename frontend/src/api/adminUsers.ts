import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type {
  AdminUser,
  DeleteUserInput,
  PaginatedAdminUsers,
  ReinstateUserInput,
  SuspendUserInput,
  UserAction,
} from "./types";

export function useAdminUsers(query: string) {
  return useQuery({
    // The query string is the key, so paging back to a view already fetched is
    // instant and two filters cannot overwrite each other's rows.
    queryKey: keys.adminUsers.list(query),
    queryFn: async () =>
      (await api.get<PaginatedAdminUsers>(`/admin/users${query ? `?${query}` : ""}`)).data,
  });
}

export function useUserHistory(userId: string) {
  return useQuery({
    queryKey: keys.adminUsers.history(userId),
    queryFn: async () =>
      (await api.get<UserAction[]>(apiPath`/admin/users/${userId}/history`)).data ?? [],
    enabled: userId !== "",
  });
}

// Every write here changes what the list shows - a status, or whether the row
// exists at all - and the history gains a row. A 409 is the same story from
// the other side: already suspended, or another admin got there first, which
// means this copy of the list is behind. So failure refetches too, for the
// same reason it does in moderation.
function invalidateAdminUsers(qc: QueryClient) {
  return qc.invalidateQueries({ queryKey: keys.adminUsers.all });
}

export function useSuspendUser() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async ({ userId, ...input }: SuspendUserInput & { userId: string }) =>
      (await api.post<AdminUser>(apiPath`/admin/users/${userId}/suspend`, input)).data,
    onSuccess: () => invalidateAdminUsers(qc),
    onError: () => invalidateAdminUsers(qc),
  });
}

export function useReinstateUser() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async ({ userId, ...input }: ReinstateUserInput & { userId: string }) =>
      (await api.post<AdminUser>(apiPath`/admin/users/${userId}/reinstate`, input)).data,
    onSuccess: () => invalidateAdminUsers(qc),
    onError: () => invalidateAdminUsers(qc),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();

  return useMutation({
    // 204, no body - unlike suspend and reinstate, which answer with the
    // account in its new state.
    mutationFn: async ({ userId, ...input }: DeleteUserInput & { userId: string }) => {
      await api.delete(apiPath`/admin/users/${userId}`, { data: input });
    },
    onSuccess: () => invalidateAdminUsers(qc),
    onError: () => invalidateAdminUsers(qc),
  });
}
