import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { api } from "../api/client";
import { useAdminOrders, useResolveOrder } from "../api/adminOrders";
import { keys } from "../api/queryKeys";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: {
    get: vi
      .fn()
      .mockResolvedValue({ data: { items: [], total: 0, page: 1, limit: 20, total_pages: 0 } }),
    post: vi.fn().mockResolvedValue({ data: {} }),
  },
}));

const ORDER_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

function invalidatedKeys(spy: { mock: { calls: unknown[][] } }) {
  return spy.mock.calls.map((call) =>
    JSON.stringify((call[0] as { queryKey?: unknown })?.queryKey),
  );
}

test("the filters travel in the query string, not just the cache key", async () => {
  const { result } = renderHook(() => useAdminOrders("stuck=true&status=confirmed"), {
    wrapper: wrapper(newClient()),
  });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(api.get).toHaveBeenCalledWith("/admin/orders?stuck=true&status=confirmed");
});

test("an unfiltered view asks for a clean path", async () => {
  const { result } = renderHook(() => useAdminOrders(""), { wrapper: wrapper(newClient()) });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(api.get).toHaveBeenCalledWith("/admin/orders");
});

// The id belongs in the path; the body carries only the outcome and the reason.
test("resolves against the order's own endpoint", async () => {
  const { result } = renderHook(() => useResolveOrder(), { wrapper: wrapper(newClient()) });

  await result.current.mutateAsync({
    orderId: ORDER_ID,
    outcome: "cancelled",
    reason: "Both parties gone",
  });

  expect(api.post).toHaveBeenCalledWith(`/admin/orders/${ORDER_ID}/resolve`, {
    outcome: "cancelled",
    reason: "Both parties gone",
  });
});

// Resolving changes the order's status and its stuck flag, and stuck is what
// the list filters on - so the row may leave the view entirely.
test("resolving refreshes the list", async () => {
  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useResolveOrder(), { wrapper: wrapper(client) });

  await result.current.mutateAsync({ orderId: ORDER_ID, outcome: "completed", reason: "why" });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));

  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminOrders.all));
});

// A 409 means the order stopped being stuck - another admin resolved it, or
// the counterparty finally acted. Either way this list is behind, so leaving
// it alone would show a row that can no longer be acted on.
test("a conflict refreshes the list rather than leaving a stale row", async () => {
  vi.mocked(api.post).mockRejectedValueOnce({ status: 409, message: "This order is not stuck" });

  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useResolveOrder(), { wrapper: wrapper(client) });

  await expect(
    result.current.mutateAsync({ orderId: ORDER_ID, outcome: "completed", reason: "why" }),
  ).rejects.toBeDefined();
  await waitFor(() => expect(result.current.isError).toBe(true));

  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminOrders.all));
});
