import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import AdminUsers from "../pages/AdminUsers";
import { api } from "../api/client";
import type { AdminUser } from "../api/types";

// Only the transport is mocked. AdminUsers.test mocks useAdminUsers itself,
// which is the right seam for the filter and URL assertions - and it is
// structurally blind to what the render does after a click, which is where
// this failure lives.
vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));
vi.mock("../components/objects/AccountActions", () => ({ AccountActions: () => null }));

const USER: AdminUser = {
  id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
  username: "forager",
  email: "f@example.com",
  role: "USER",
  status: "active",
  created_at: "2026-08-01T00:00:00Z",
};

function respondSlowly() {
  vi.mocked(api.get).mockImplementation(
    () =>
      new Promise((resolve) =>
        setTimeout(
          () => resolve({ data: { items: [USER], total: 40, page: 1, limit: 20, total_pages: 2 } }),
          50,
        ),
      ) as never,
  );
}

function renderPage() {
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter initialEntries={["/admin/users"]}>
        <AdminUsers />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// A page change is a new query key. Without keepPreviousData the list and the
// pager unmount while the next page loads, so the control the reader just
// clicked disappears and focus falls to <body> - a keyboard admin has to tab
// in from the top of the document to reach page 3.
test("paging keeps the list, the pager and the focus", async () => {
  respondSlowly();
  renderPage();

  const next = await screen.findByRole("button", { name: "Next" });
  next.focus();
  await userEvent.click(next);

  // Mid-flight, before the second page lands.
  expect(screen.getByRole("navigation", { name: "Pagination" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  expect(screen.getAllByRole("listitem")).toHaveLength(1);
  expect(document.activeElement?.tagName).toBe("BUTTON");

  await waitFor(() => expect(screen.getAllByRole("listitem")).toHaveLength(1));
});

// The placeholder is dropped on error, so a failed page surfaces rather than
// silently leaving the reader on the previous page's rows believing they moved.
test("a failed page says so rather than showing the previous one", async () => {
  vi.mocked(api.get)
    .mockResolvedValueOnce({
      data: { items: [USER], total: 40, page: 1, limit: 20, total_pages: 2 },
    } as never)
    .mockRejectedValueOnce({ status: 500, message: "Something went wrong" });

  renderPage();

  await userEvent.click(await screen.findByRole("button", { name: "Next" }));

  expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
  expect(screen.queryByText("forager")).not.toBeInTheDocument();
});
