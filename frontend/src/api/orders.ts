import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { Order } from "./types";

// The backend sends order prices as strings ("18.00") while listing price is a
// number. Number() handles both, so this survives #120 and becomes deletable
// rather than breaking when it lands.
type OrderWire = Omit<Order, "unit_price" | "total_price"> & {
  unit_price: string | number;
  total_price: string | number;
};

function toOrder(o: OrderWire): Order {
  return { ...o, unit_price: Number(o.unit_price), total_price: Number(o.total_price) };
}

export interface CreateOrderInput {
  listing_id: string;
  quantity: number;
}

// --- Reads ---

export function useOrders(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: keys.orders.list(),
    queryFn: async () => ((await api.get<OrderWire[]>("/orders")).data ?? []).map(toOrder),
    // Signed-out callers pass false: the request would 401 and burn a refresh.
    enabled: options.enabled ?? true,
  });
}

export function useOrder(id: string) {
  return useQuery({
    queryKey: keys.orders.detail(id),
    queryFn: async () => toOrder((await api.get<OrderWire>(apiPath`/orders/${id}`)).data),
    enabled: id !== "",
  });
}

// --- Writes ---

export function useCreateOrder() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateOrderInput) =>
      toOrder((await api.post<OrderWire>("/orders", input)).data),

    onSuccess: (order) => {
      qc.setQueryData(keys.orders.detail(order.id), order);
      qc.invalidateQueries({ queryKey: keys.orders.all });
      // Ordering reserves stock, so every cached listing is now wrong.
      qc.invalidateQueries({ queryKey: keys.listings.all });
    },
  });
}

// The four transitions differ only by path and by whether they move stock.
type Transition = "confirm" | "handover" | "receive" | "cancel";

function useOrderTransition(action: Transition) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) =>
      toOrder((await api.post<OrderWire>(apiPath`/orders/${id}/${action}`)).data),

    onSuccess: (order) => {
      // The response IS the new state, so write it in rather than refetching.
      qc.setQueryData(keys.orders.detail(order.id), order);
      qc.invalidateQueries({ queryKey: keys.orders.all });

      // Only cancelling returns stock to the listing.
      if (action === "cancel") {
        qc.invalidateQueries({ queryKey: keys.listings.all });
      }
    },
  });
}

export const useConfirmOrder = () => useOrderTransition("confirm");
export const useHandoverOrder = () => useOrderTransition("handover");
export const useReceiveOrder = () => useOrderTransition("receive");
export const useCancelOrder = () => useOrderTransition("cancel");
