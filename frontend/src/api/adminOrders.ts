import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { AdminOrder, PaginatedAdminOrders, ResolveOrderInput } from "./types";

export function useAdminOrders(query: string, options: { enabled?: boolean } = {}) {
  return useQuery({
    // The query string is the key, so paging back to a view already fetched is
    // instant and two filters cannot overwrite each other's rows.
    queryKey: keys.adminOrders.list(query),
    // No price normalising here, unlike orders.ts: OrderResponse types
    // unit_price and total_price as float64 since #120, so they arrive as
    // JSON numbers. Copying that workaround would carry a fix for a problem
    // that no longer exists.
    queryFn: async () =>
      (await api.get<PaginatedAdminOrders>(`/admin/orders${query ? `?${query}` : ""}`)).data,
    // The page passes false for a range the API would answer with a 400, so a
    // request certain to fail is never sent.
    enabled: options.enabled ?? true,
    // A page change is a new query key, so without this the list and the pager
    // unmount mid-click and take keyboard focus to <body> with them. The
    // placeholder is dropped on error, so a failed page still surfaces.
    placeholderData: keepPreviousData,
  });
}

export function useResolveOrder() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async ({ orderId, ...input }: ResolveOrderInput & { orderId: string }) =>
      (await api.post<AdminOrder>(apiPath`/admin/orders/${orderId}/resolve`, input)).data,

    // The order's status and its stuck flag both change, and stuck is what the
    // list is filtered by - so the row may leave the view entirely.
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.adminOrders.all }),

    // A 409 is "this order is not stuck any more", which usually means another
    // admin resolved it or the counterparty finally acted. Either way this copy
    // of the list is behind, so a failure refreshes it too.
    onError: () => qc.invalidateQueries({ queryKey: keys.adminOrders.all }),
  });
}
