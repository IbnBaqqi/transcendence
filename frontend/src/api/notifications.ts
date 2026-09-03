import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "./client";
import { keys } from "./queryKeys";
import type { Notification } from "./types";

// Demo-scale polling like the inbox beside it: no WebSockets in scope (#88).
const POLL_MS = 30_000;

export function useNotifications(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: keys.me.notifications(),
    queryFn: async () => (await api.get<Notification[]>("/me/notifications")).data ?? [],
    refetchInterval: POLL_MS,
    enabled: options.enabled ?? true,
  });
}

export function unreadCount(notifications: Notification[] | undefined) {
  return notifications?.filter((n) => n.read_at === null).length ?? 0;
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await api.post("/me/notifications/read");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.me.notifications() }),
  });
}
