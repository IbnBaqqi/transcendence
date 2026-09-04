import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import AdminOrders from "../pages/AdminOrders";
import { useAdminOrders } from "../api/adminOrders";
import type { AdminOrder, PaginatedAdminOrders } from "../api/types";

vi.mock("../api/adminOrders", () => ({ useAdminOrders: vi.fn() }));

const ORDER_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

function makeOrder(overrides: Partial<AdminOrder> = {}): AdminOrder {
  return {
    id: ORDER_ID,
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
    ...overrides,
  };
}

function page(items: AdminOrder[], total = items.length, over: Partial<PaginatedAdminOrders> = {}) {
  return { items, total, page: 1, limit: 20, total_pages: 1, ...over };
}

const loaded = (
  items: AdminOrder[],
  total = items.length,
  over: Partial<PaginatedAdminOrders> = {},
) => ({ data: page(items, total, over), isPending: false as const, isError: false as const });

function ShowUrl() {
  return <span data-testid="url">{useLocation().search}</span>;
}
const url = () => screen.getByTestId("url").textContent;

function renderPage(query: Partial<ReturnType<typeof useAdminOrders>>, at = "/admin/orders") {
  vi.mocked(useAdminOrders).mockReturnValue(query as ReturnType<typeof useAdminOrders>);
  render(
    <MemoryRouter initialEntries={[at]}>
      <Routes>
        <Route
          path="/admin/orders"
          element={
            <>
              <AdminOrders />
              <ShowUrl />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

// Scoped to the row: the status filter has an option of the same name, so an
// unscoped query matches the control as well as the pill.
const row = () => within(screen.getByRole("listitem"));

test("lists the orders it is given", () => {
  renderPage(loaded([makeOrder()]));
  expect(screen.getByRole("link", { name: "Golden Chanterelles" })).toBeInTheDocument();
  expect(row().getByText("Confirmed")).toBeInTheDocument();
  expect(row().getByText("2 × — 37 total")).toBeInTheDocument();
});

// The reason the screen exists: an order no party can move.
test("marks a stuck order", () => {
  renderPage(loaded([makeOrder({ stuck: true })]));
  expect(row().getByText("Nobody can move this")).toBeInTheDocument();
});

test("does not mark an order anyone can still move", () => {
  renderPage(loaded([makeOrder({ stuck: false })]));
  expect(screen.queryByText("Nobody can move this")).not.toBeInTheDocument();
});

// Exactly one mark is what trapped mid-handover looks like, so the row says
// which side acted - that is who the order is waiting on.
test.each([
  [{ seller_handed_over_at: "2026-08-02T00:00:00Z" }, "Seller handed over"],
  [{ buyer_received_at: "2026-08-02T00:00:00Z" }, "Buyer received"],
  [
    { seller_handed_over_at: "2026-08-02T00:00:00Z", buyer_received_at: "2026-08-03T00:00:00Z" },
    "Both sides acted",
  ],
  [{}, "Neither side has acted"],
])("says which side has acted", (marks, expected) => {
  renderPage(loaded([makeOrder(marks)]));
  expect(row().getByText(expected)).toBeInTheDocument();
});

describe("filters", () => {
  test("choosing a status writes it to the URL", async () => {
    renderPage(loaded([makeOrder()]));
    await userEvent.selectOptions(screen.getByLabelText("Status"), "confirmed");
    expect(url()).toBe("?status=confirmed");
  });

  // Three states, not a checkbox: "any" and "not stuck" are different queries.
  test.each([
    ["true", "?stuck=true"],
    ["false", "?stuck=false"],
  ])("stuck=%s round-trips through the URL", async (value, expected) => {
    renderPage(loaded([makeOrder()]));
    await userEvent.selectOptions(screen.getByLabelText("Stuck"), value);
    expect(url()).toBe(expected);
  });

  test("the controls show what the URL already says", () => {
    renderPage(loaded([makeOrder()]), "/admin/orders?status=refunded&stuck=false");
    expect(screen.getByLabelText("Status")).toHaveValue("refunded");
    expect(screen.getByLabelText("Stuck")).toHaveValue("false");
  });

  // The API answers a reversed range with a 400, which reads as a server fault
  // rather than a typo.
  test("says so when the range ends before it starts", () => {
    renderPage(
      loaded([makeOrder()]),
      "/admin/orders?created_from=2026-08-31&created_to=2026-08-01",
    );
    expect(screen.getByRole("alert")).toHaveTextContent("The date range ends before it starts.");
  });

  test("says nothing about a range that runs forwards", () => {
    renderPage(
      loaded([makeOrder()]),
      "/admin/orders?created_from=2026-08-01&created_to=2026-08-31",
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  // The issue names this one: a date range with no matches is an empty state,
  // not an error.
  test("an empty result says nothing matches rather than erroring", () => {
    renderPage(loaded([], 0), "/admin/orders?created_from=2026-08-01&created_to=2026-08-02");
    expect(screen.getByText("No orders match those filters.")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});

// The two serialisers are not interchangeable and this is the assertion that
// says so: ToParams writes the URL in the dates the admin picked, ToQuery
// converts them to the instants the API means. Passing the URL's own string to
// the hook compiles, reads as a tidy simplification, and silently sends UTC
// midnights - shifting every range by the local offset and dropping the last
// day entirely.
describe("what the page asks the API for", () => {
  const sentQuery = () =>
    new URLSearchParams(vi.mocked(useAdminOrders).mock.calls.at(-1)?.[0] ?? "");

  test("converts the picked dates into instants", () => {
    renderPage(
      loaded([makeOrder()]),
      "/admin/orders?created_from=2026-08-01&created_to=2026-08-31",
    );

    // Local midnight on both sides of the comparison, so this holds in any zone.
    expect(sentQuery().get("created_from")).toBe(new Date(2026, 7, 1).toISOString());
    // Exclusive: the whole of 31 August means everything before 1 September.
    expect(sentQuery().get("created_to")).toBe(new Date(2026, 8, 1).toISOString());
  });

  test("never sends a bare date", () => {
    renderPage(loaded([makeOrder()]), "/admin/orders?created_from=2026-08-01");
    expect(sentQuery().get("created_from")).not.toBe("2026-08-01");
  });

  test("passes the other filters through untouched", () => {
    renderPage(loaded([makeOrder()]), "/admin/orders?status=refunded&stuck=true&page=3");
    const sent = sentQuery();
    expect(sent.get("status")).toBe("refunded");
    expect(sent.get("stuck")).toBe("true");
    expect(sent.get("page")).toBe("3");
  });
});

describe("paging", () => {
  test("keeps every filter when the page changes", async () => {
    renderPage(
      loaded([makeOrder()], 40, { page: 1, total_pages: 2 }),
      "/admin/orders?status=confirmed&stuck=true&created_from=2026-08-01",
    );

    await userEvent.click(screen.getByRole("button", { name: "Next" }));

    expect(url()).toBe("?status=confirmed&stuck=true&created_from=2026-08-01&page=2");
  });

  test("offers no pager when nothing matches", () => {
    renderPage(loaded([], 0));
    expect(screen.queryByRole("navigation", { name: "Pagination" })).not.toBeInTheDocument();
  });
});
