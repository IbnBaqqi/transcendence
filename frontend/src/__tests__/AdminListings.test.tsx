import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import AdminListings from "../pages/AdminListings";
import { useReportQueue } from "../api/moderation";
import type { ReportedListing } from "../api/types";

vi.mock("../api/moderation", () => ({ useReportQueue: vi.fn() }));

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

function makeRow(overrides: Partial<ReportedListing> = {}): ReportedListing {
  return {
    listing_id: LISTING_ID,
    title: "Golden Chanterelles",
    seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    removed_at: null,
    report_count: 3,
    first_reported_at: "2026-08-01T00:00:00Z",
    ...overrides,
  };
}

function renderQueue(query: Partial<ReturnType<typeof useReportQueue>>) {
  vi.mocked(useReportQueue).mockReturnValue(query as ReturnType<typeof useReportQueue>);
  render(
    <MemoryRouter>
      <AdminListings />
    </MemoryRouter>,
  );
}

test("says so when the queue is empty", () => {
  renderQueue({ data: [], isPending: false, isError: false });
  expect(screen.getByText("Nothing waiting. Every report has been decided.")).toBeInTheDocument();
});

// report_count leads the row: three complaints about one listing is one
// problem, and the count is what decides which row to open first.
test("leads each row with its report count", () => {
  renderQueue({ data: [makeRow({ report_count: 3 })], isPending: false, isError: false });
  expect(screen.getByText("3 reports")).toBeInTheDocument();
});

test("counts a single report in the singular", () => {
  renderQueue({ data: [makeRow({ report_count: 1 })], isPending: false, isError: false });
  expect(screen.getByText("1 report")).toBeInTheDocument();
});

test("links each row to its listing", () => {
  renderQueue({ data: [makeRow()], isPending: false, isError: false });
  expect(screen.getByRole("link", { name: "Golden Chanterelles" })).toHaveAttribute(
    "href",
    `/listings/${LISTING_ID}`,
  );
});

// removed_at is what separates a listing still up from one already taken
// down - the row that decides whether the action is remove or restore.
test("marks a listing that is already removed", () => {
  renderQueue({
    data: [makeRow({ removed_at: "2026-08-02T00:00:00Z" })],
    isPending: false,
    isError: false,
  });
  expect(screen.getByText("Removed")).toBeInTheDocument();
});

test("does not mark a listing that is still up", () => {
  renderQueue({ data: [makeRow()], isPending: false, isError: false });
  expect(screen.queryByText("Removed")).not.toBeInTheDocument();
});

// The server already returns oldest complaint first (ORDER BY min(created_at)),
// so the page must render them in the order it was given rather than sorting
// again and holding a second opinion about priority.
test("renders the queue in the order the server sent it", () => {
  renderQueue({
    data: [
      makeRow({
        listing_id: "a",
        title: "Oldest complaint",
        first_reported_at: "2026-08-01T00:00:00Z",
      }),
      makeRow({
        listing_id: "b",
        title: "Newer complaint",
        first_reported_at: "2026-08-09T00:00:00Z",
      }),
    ],
    isPending: false,
    isError: false,
  });
  const links = screen.getAllByRole("link");
  expect(links[0]).toHaveTextContent("Oldest complaint");
  expect(links[1]).toHaveTextContent("Newer complaint");
});

test("offers a retry when the queue will not load", async () => {
  const refetch = vi.fn();
  renderQueue({
    data: undefined,
    isPending: false,
    isError: true,
    error: new Error("boom"),
    refetch,
  });
  expect(screen.getByText("Couldn't load the moderation queue.")).toBeInTheDocument();

  await userEvent.click(screen.getByRole("button", { name: "Try again" }));
  expect(refetch).toHaveBeenCalled();
});
