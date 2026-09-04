import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import AdminOrders from "../pages/AdminOrders";
import { api } from "../api/client";
import type { AdminOrder } from "../api/types";

// Only the transport is mocked. AdminOrders.test mocks useAdminOrders, which
// is the right seam for the filter and URL assertions and is structurally
// blind to whether a request goes out and to the render that follows a click.
vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { get: vi.fn(), post: vi.fn() },
}));
vi.mock("../components/objects/ResolveOrderDialog", () => ({ ResolveOrderDialog: () => null }));

const ORDER: AdminOrder = {
  id: "01a02305-b81c-7dcb-86a0-7f75e33e0af3",
  listing_id: "01a02305-b81c-7dcb-86a0-7f75e33e0af4",
  listing_title: "Golden Chanterelles",
  buyer_id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
  seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
  quantity: 2,
  unit_price: 18.5,
  total_price: 37,
  status: "confirmed",
  seller_handed_over_at: null,
  buyer_received_at: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  stuck: false,
};

function renderAt(url = "/admin/orders") {
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter initialEntries={[url]}>
        <AdminOrders />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// The API answers a reversed range with a 400. Sending it anyway shows the
// admin both messages at once - the friendly one and the raw server error,
// with a Try again button that fails identically every time.
test("a backwards range never leaves the client", async () => {
  vi.mocked(api.get).mockRejectedValue({
    status: 400,
    message: "The date range ends before it starts",
  });

  renderAt("/admin/orders?created_from=2026-09-01&created_to=2026-08-01");
  await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

  expect(api.get).not.toHaveBeenCalled();
  expect(screen.queryByText("The date range ends before it starts")).not.toBeInTheDocument();
});

// A disabled query reports isPending forever, so gating the request without
// gating the skeletons leaves three of them spinning under the alert.
test("a backwards range does not sit under spinning skeletons", async () => {
  renderAt("/admin/orders?created_from=2026-09-01&created_to=2026-08-01");
  await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

  // The loading variant of Skeleton is the only thing carrying this class -
  // its error variant does not, so this counts loading placeholders alone.
  expect(document.querySelectorAll(".skeleton")).toHaveLength(0);
});

// The control: the same page with a range that runs forwards does show them
// while the request is in flight, so the assertion above is not vacuous.
test("a forwards range does show skeletons while it loads", async () => {
  vi.mocked(api.get).mockImplementation(() => new Promise(() => {}) as never);

  renderAt("/admin/orders?created_from=2026-08-01&created_to=2026-08-31");

  await waitFor(() => expect(document.querySelectorAll(".skeleton").length).toBeGreaterThan(0));
});

test("a range that runs forwards is sent", async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: { items: [ORDER], total: 1, page: 1, limit: 20, total_pages: 1 },
  } as never);

  renderAt("/admin/orders?created_from=2026-08-01&created_to=2026-08-31");

  await waitFor(() => expect(api.get).toHaveBeenCalled());
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});

// A page change is a new query key: without keepPreviousData the list and the
// pager unmount while the next page loads, so the control just clicked
// disappears and focus falls to <body>.
test("paging keeps the list, the pager and the focus", async () => {
  vi.mocked(api.get).mockImplementation(
    () =>
      new Promise((resolve) =>
        setTimeout(
          () =>
            resolve({ data: { items: [ORDER], total: 40, page: 1, limit: 20, total_pages: 2 } }),
          50,
        ),
      ) as never,
  );

  renderAt();

  const next = await screen.findByRole("button", { name: "Next" });
  next.focus();
  await userEvent.click(next);

  expect(screen.getByRole("navigation", { name: "Pagination" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  expect(screen.getAllByRole("listitem")).toHaveLength(1);
  expect(document.activeElement?.tagName).toBe("BUTTON");

  await waitFor(() => expect(screen.getAllByRole("listitem")).toHaveLength(1));
});
