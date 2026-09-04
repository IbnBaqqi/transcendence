import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { api } from "../api/client";
import { useAdminUsers, useDeleteUser, useReinstateUser, useSuspendUser } from "../api/adminUsers";
import { keys } from "../api/queryKeys";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: {
    get: vi
      .fn()
      .mockResolvedValue({ data: { items: [], total: 0, page: 1, limit: 20, total_pages: 0 } }),
    post: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: undefined }),
  },
}));

const USER_ID = "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6";

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function invalidatedKeys(spy: { mock: { calls: unknown[][] } }) {
  return spy.mock.calls.map((call) =>
    JSON.stringify((call[0] as { queryKey?: unknown })?.queryKey),
  );
}

test("the filters travel in the query string, not just the cache key", async () => {
  const { result } = renderHook(() => useAdminUsers("status=suspended&page=2"), {
    wrapper: wrapper(newClient()),
  });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(api.get).toHaveBeenCalledWith("/admin/users?status=suspended&page=2");
});

test("an unfiltered view asks for a clean path", async () => {
  const { result } = renderHook(() => useAdminUsers(""), { wrapper: wrapper(newClient()) });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(api.get).toHaveBeenCalledWith("/admin/users");
});

// axios sends a DELETE body only under `data`. Passing the object directly
// compiles, is treated as request config, and silently sends nothing - so the
// server sees no username and answers 400 for a confirmation the admin typed
// correctly.
test("delete sends its confirmation as a body", async () => {
  const { result } = renderHook(() => useDeleteUser(), { wrapper: wrapper(newClient()) });

  await result.current.mutateAsync({ userId: USER_ID, username: "forager", reason: "spam" });

  expect(api.delete).toHaveBeenCalledWith(`/admin/users/${USER_ID}`, {
    data: { username: "forager", reason: "spam" },
  });
});

test("suspend and reinstate put the id in the path and the rest in the body", async () => {
  const client = newClient();
  const suspend = renderHook(() => useSuspendUser(), { wrapper: wrapper(client) });
  await suspend.result.current.mutateAsync({ userId: USER_ID, reason: "Listing items they lack" });
  expect(api.post).toHaveBeenCalledWith(`/admin/users/${USER_ID}/suspend`, {
    reason: "Listing items they lack",
  });

  const reinstate = renderHook(() => useReinstateUser(), { wrapper: wrapper(client) });
  await reinstate.result.current.mutateAsync({ userId: USER_ID });
  expect(api.post).toHaveBeenCalledWith(`/admin/users/${USER_ID}/reinstate`, {});
});

// Every write changes what the list shows - a status, or whether the row is
// there at all - so each one has to refresh it.
test("suspending refreshes the list", async () => {
  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useSuspendUser(), { wrapper: wrapper(client) });

  await result.current.mutateAsync({ userId: USER_ID, reason: "why" });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminUsers.all));
});

test("reinstating refreshes the list", async () => {
  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useReinstateUser(), { wrapper: wrapper(client) });

  await result.current.mutateAsync({ userId: USER_ID });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminUsers.all));
});

test("deleting refreshes the list", async () => {
  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useDeleteUser(), { wrapper: wrapper(client) });

  await result.current.mutateAsync({ userId: USER_ID, username: "forager", reason: "why" });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminUsers.all));
});

// A 409 is "already suspended" or "another admin got there first" - both mean
// this copy of the list is behind, so a failure has to refresh it rather than
// leaving a row whose status is wrong.
test("a conflict refreshes the list rather than leaving a stale row", async () => {
  vi.mocked(api.post).mockRejectedValueOnce({ status: 409, message: "Already suspended" });

  const client = newClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderHook(() => useSuspendUser(), { wrapper: wrapper(client) });

  await expect(
    result.current.mutateAsync({ userId: USER_ID, reason: "why" }),
  ).rejects.toBeDefined();
  await waitFor(() => expect(result.current.isError).toBe(true));

  expect(invalidatedKeys(invalidate)).toContain(JSON.stringify(keys.adminUsers.all));
});
