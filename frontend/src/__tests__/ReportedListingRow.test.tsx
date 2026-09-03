import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { ReportedListingRow } from "../components/objects/ReportedListingRow";
import { useListingReports, useModerationHistory } from "../api/moderation";
import type { ModerationAction, Report, ReportedListing } from "../api/types";

vi.mock("../api/moderation", () => ({
  useListingReports: vi.fn(),
  useModerationHistory: vi.fn(),
}));

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

const ROW: ReportedListing = {
  listing_id: LISTING_ID,
  title: "Golden Chanterelles",
  seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
  removed_at: null,
  report_count: 2,
  first_reported_at: "2026-08-01T00:00:00Z",
};

function makeReport(overrides: Partial<Report> = {}): Report {
  return {
    id: "r1",
    reporter_id: "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6",
    reason: "spam",
    detail: "Posted the same thing eight times",
    status: "open",
    created_at: "2026-08-01T00:00:00Z",
    ...overrides,
  };
}

function makeAction(overrides: Partial<ModerationAction> = {}): ModerationAction {
  return {
    id: "m1",
    listing_id: LISTING_ID,
    moderator_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    action: "dismissed",
    note: "Looked fine",
    created_at: "2026-08-02T00:00:00Z",
    ...overrides,
  };
}

function setData(reports: Report[] | undefined, history: ModerationAction[] | undefined) {
  vi.mocked(useListingReports).mockReturnValue({ data: reports } as ReturnType<
    typeof useListingReports
  >);
  vi.mocked(useModerationHistory).mockReturnValue({ data: history } as ReturnType<
    typeof useModerationHistory
  >);
}

function renderRow(row: ReportedListing = ROW) {
  render(
    <MemoryRouter>
      <ul>
        <ReportedListingRow row={row} />
      </ul>
    </MemoryRouter>,
  );
}

beforeEach(() => setData([makeReport()], [makeAction()]));

// A collapsed row passes "" so both queries stay disabled - that is what makes
// rendering a long queue cost nothing beyond the queue itself.
test("asks for nothing until the row is opened", () => {
  renderRow();
  expect(useListingReports).toHaveBeenCalledWith("");
  expect(useModerationHistory).toHaveBeenCalledWith("");
});

test("asks for that listing once opened", async () => {
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(useListingReports).toHaveBeenCalledWith(LISTING_ID);
  expect(useModerationHistory).toHaveBeenCalledWith(LISTING_ID);
});

test("keeps the detail shut until asked", () => {
  renderRow();
  expect(screen.queryByText("Posted the same thing eight times")).not.toBeInTheDocument();
});

test("shows the reports and the history side by side", async () => {
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(screen.getByText("Spam")).toBeInTheDocument();
  expect(screen.getByText("Posted the same thing eight times")).toBeInTheDocument();
  expect(screen.getByText("Looked fine")).toBeInTheDocument();
});

// Resolved reports are context, not noise: a listing reported before and
// cleared reads differently from a fresh one.
test("keeps resolved reports visible rather than filtering them out", async () => {
  setData([makeReport({ id: "r1", status: "dismissed", detail: "An old complaint" })], []);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(screen.getByText("An old complaint")).toBeInTheDocument();
});

// The report outlives its author on purpose - the complaint is about the
// listing, not the person.
test("says so when the reporter has deleted their account", async () => {
  setData([makeReport({ reporter_id: null })], []);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(screen.getByText("Reporter has deleted their account")).toBeInTheDocument();
});

test("says so when the moderator has deleted their account", async () => {
  setData([], [makeAction({ moderator_id: null })]);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(screen.getByText("Moderator has deleted their account")).toBeInTheDocument();
});

test("says a listing has never been actioned", async () => {
  setData([makeReport()], []);
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  expect(screen.getByText("Never actioned.")).toBeInTheDocument();
});

test("renders both lists in the order the server sent them", async () => {
  setData(
    [
      makeReport({ id: "r1", detail: "Newest complaint" }),
      makeReport({ id: "r2", detail: "Older complaint" }),
    ],
    [],
  );
  renderRow();
  await userEvent.click(screen.getByRole("button", { name: "Show detail" }));
  const details = screen.getAllByText(/complaint$/);
  expect(details[0]).toHaveTextContent("Newest complaint");
  expect(details[1]).toHaveTextContent("Older complaint");
});
