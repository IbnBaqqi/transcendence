import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { api } from "../api/client";
import { useModerateListing } from "../api/moderation";
import { keys } from "../api/queryKeys";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { post: vi.fn().mockResolvedValue({ data: { listing: {}, reports_resolved: 3 } }) },
}));

const post = vi.mocked(api.post);

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

function renderModerate(client: QueryClient) {
  return renderHook(() => useModerateListing(), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client }, children),
  });
}

// The listing id belongs in the path, not the body: the endpoint is
// /admin/listings/{id}/moderate and the body is only { action, note }.
test("sends the action to the listing's own endpoint", async () => {
  const { result } = renderModerate(new QueryClient());

  await result.current.mutateAsync({ listingId: LISTING_ID, action: "remove", note: "spam" });

  expect(api.post).toHaveBeenCalledWith(`/admin/listings/${LISTING_ID}/moderate`, {
    action: "remove",
    note: "spam",
  });
});

// One action resolves every open report on the listing and writes an audit
// row, so the queue, that listing's reports and its history are all stale -
// and the listing's own visibility changed too. A key missing here fails
// silently: the screen keeps showing what it already had.
test("moderating invalidates everything one decision changes", async () => {
  const client = new QueryClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");

  const { result } = renderModerate(client);

  await result.current.mutateAsync({ listingId: LISTING_ID, action: "dismiss" });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));

  const invalidated = invalidate.mock.calls.map(([arg]) => JSON.stringify(arg?.queryKey));
  for (const key of [keys.moderation.all, keys.listings.all]) {
    expect(invalidated).toContain(JSON.stringify(key));
  }

  // listings.all is enough for the detail entry too: keys match by prefix, and
  // detail(id) is ["listings", "detail", id]. Asserting both would pin the
  // implementation rather than the behaviour, which is "the listing is stale".
  expect(keys.listings.detail(LISTING_ID).slice(0, keys.listings.all.length)).toEqual([
    ...keys.listings.all,
  ]);
});

// Prefix matching is the whole reason moderation.all is enough: it has to
// cover the two per-listing keys, not just the queue.
test("the moderation prefix covers the per-listing keys", () => {
  const prefix = JSON.stringify(keys.moderation.all);
  for (const key of [
    keys.moderation.queue(),
    keys.moderation.reports(LISTING_ID),
    keys.moderation.history(LISTING_ID),
  ]) {
    expect(JSON.stringify(key).startsWith(prefix.slice(0, -1))).toBe(true);
  }
});

// A failure means this copy of the moderation data is out of date: a 409 is
// usually another moderator getting there first, a 404 the listing being gone.
// Their decision also changed the reports and history sitting open on screen,
// so a failure has to refresh all three, not just the queue.
test("refetches everything moderation when a decision fails", async () => {
  post.mockRejectedValueOnce({ status: 409, message: "Already removed" });

  const client = new QueryClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const { result } = renderModerate(client);

  await expect(
    result.current.mutateAsync({ listingId: LISTING_ID, action: "remove", note: "spam" }),
  ).rejects.toBeDefined();
  await waitFor(() => expect(result.current.isError).toBe(true));

  const invalidated = invalidate.mock.calls.map(([arg]) => JSON.stringify(arg?.queryKey));
  expect(invalidated).toContain(JSON.stringify(keys.moderation.all));
});
