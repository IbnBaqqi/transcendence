import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { useBlock } from "../api/blocks";
import { keys } from "../api/queryKeys";
import { SELLER_ID } from "../test/factories";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { post: vi.fn().mockResolvedValue({ data: undefined }) },
}));

// A block changes more than the block list: it hides presence in either
// direction, which the follow lists fold into show_online_status in SQL, and
// closes the thread. A key missing here fails silently - the screen simply
// keeps showing what it already had.
test("blocking invalidates everything a block changes", async () => {
  const client = new QueryClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");

  const { result } = renderHook(() => useBlock(), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client }, children),
  });

  await result.current.mutateAsync(SELLER_ID);
  await waitFor(() => expect(result.current.isSuccess).toBe(true));

  const invalidated = invalidate.mock.calls.map(([arg]) => JSON.stringify(arg?.queryKey));
  for (const key of [
    keys.me.blocks(),
    keys.users.detail(SELLER_ID),
    keys.conversations.all,
    keys.follows.all,
  ]) {
    expect(invalidated).toContain(JSON.stringify(key));
  }
});
