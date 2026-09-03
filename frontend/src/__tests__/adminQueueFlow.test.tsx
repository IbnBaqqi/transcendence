import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import AdminListings from "../pages/AdminListings";
import { api } from "../api/client";
import type { ReportedListing } from "../api/types";

// Only the transport is mocked here. The hooks, their cache keys and their
// invalidation are the thing under test: the queue and the decision are
// pinned apart elsewhere, and this is what asserts they are connected.
vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { get: vi.fn(), post: vi.fn() },
}));

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

const ROW: ReportedListing = {
  listing_id: LISTING_ID,
  title: "Golden Chanterelles",
  seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
  removed_at: null,
  report_count: 3,
  first_reported_at: "2026-08-01T00:00:00Z",
};

function renderQueue() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AdminListings />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// One moderate call resolves every open report on the listing, so the row
// leaves the queue. Nothing in the UI removes it - the refetch does, and this
// is the assertion that the two are wired together.
test("deciding clears the row from the queue without a reload", async () => {
  const get = vi.mocked(api.get);
  const post = vi.mocked(api.post);

  // The queue has the row, then does not. Detail calls answer empty.
  get.mockImplementation((url: string) => {
    if (url === "/admin/reports") {
      return Promise.resolve({ data: post.mock.calls.length === 0 ? [ROW] : [] });
    }
    return Promise.resolve({ data: [] });
  });
  post.mockResolvedValue({ data: { listing: {}, reports_resolved: 3 } });

  renderQueue();
  expect(await screen.findByText("Golden Chanterelles")).toBeInTheDocument();

  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  await userEvent.click(screen.getByRole("button", { name: "Dismiss" }));
  await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

  await waitFor(() => {
    expect(screen.queryByText("Golden Chanterelles")).not.toBeInTheDocument();
  });
  expect(screen.getByText("Nothing waiting. Every report has been decided.")).toBeInTheDocument();
});
