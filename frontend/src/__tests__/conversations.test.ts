import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { api } from "../api/client";
import { useMessages } from "../api/conversations";
import type { Message } from "../api/types";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { get: vi.fn() },
}));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return createElement(QueryClientProvider, { client }, children);
}

function makeMessage(id: string, body: string): Message {
  return { id, conversation_id: "c1", sender_id: "u1", body, created_at: "1970-01-01T00:00:00Z" };
}

// MessageThread mocks useMessages, so nothing covered the transform itself -
// which is where the bug was. The SQL is ORDER BY id DESC, but ListMessages
// calls slices.Reverse before responding, so the API already sends
// oldest-first and sorting here renders every thread backwards.
test("useMessages returns messages in the order the API sent them", async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: [makeMessage("m1", "oldest"), makeMessage("m2", "newest")],
  } as Awaited<ReturnType<typeof api.get>>);

  const { result } = renderHook(() => useMessages("c1"), { wrapper });

  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(result.current.data?.map((m) => m.body)).toEqual(["oldest", "newest"]);
});
