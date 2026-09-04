import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ReportListingSection } from "../components/forms/ReportListingSection";
import { useListing, useReportListing } from "../api/listings";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { makeListing, BUYER_ID, SELLER_ID } from "../test/factories";
import type { User } from "../api/types";

vi.mock("../api/listings", () => ({ useListing: vi.fn(), useReportListing: vi.fn() }));

const LISTING_ID = "01a02305-b81c-7dcb-86a0-7f75e33e0af3";

const mutate = vi.fn();

const VIEWER: User = {
  id: BUYER_ID,
  username: "tester",
  email: "t@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

function setReport(overrides: Record<string, unknown> = {}) {
  vi.mocked(useReportListing).mockReturnValue({
    mutate,
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
    ...overrides,
  } as unknown as ReturnType<typeof useReportListing>);
}

beforeEach(() => {
  mutate.mockReset();
  setReport();
});

function authStub(user: User | null): AuthContextValue {
  return {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };
}

// makeListing's default seller_id is SELLER_ID, so the viewer is a stranger to
// the listing unless a test says otherwise.
function renderSection(user: User | null = VIEWER) {
  vi.mocked(useListing).mockReturnValue({
    data: makeListing(),
    isPending: false,
  } as ReturnType<typeof useListing>);

  return render(
    <AuthContext.Provider value={authStub(user)}>
      <ReportListingSection listingId={LISTING_ID} />
    </AuthContext.Provider>,
  );
}

// The API refuses a report on your own listing with a 400, so offering the
// control would be a guaranteed error rather than a choice.
test("offers nothing to the listing's own seller", () => {
  renderSection({ ...VIEWER, id: SELLER_ID });
  expect(screen.queryByRole("button", { name: "Report this listing" })).not.toBeInTheDocument();
});

// Mirrors the server: every other reason states the problem by itself, while
// "other" says only that none of them fit. Whitespace does not count there
// either - the server trims before it checks.
test('holds an "other" report until it has a detail, then sends it', async () => {
  renderSection();
  await userEvent.click(screen.getByRole("button", { name: "Report this listing" }));
  await userEvent.click(screen.getByRole("radio", { name: "Other" }));

  expect(screen.getByRole("button", { name: "Send report" })).toBeDisabled();

  await userEvent.type(screen.getByRole("textbox"), "   ");
  expect(screen.getByRole("button", { name: "Send report" })).toBeDisabled();

  await userEvent.type(screen.getByRole("textbox"), "Wrong species");
  await userEvent.click(screen.getByRole("button", { name: "Send report" }));

  expect(mutate).toHaveBeenCalledWith({
    listingId: LISTING_ID,
    reason: "other",
    detail: "Wrong species",
  });
});

// One report per person per listing, so a second attempt is a 409 - which
// means the complaint is already on record. That is what the reporter wanted,
// so it ends the flow instead of turning red.
test("treats a 409 as the report already being on record", () => {
  setReport({
    isError: true,
    error: { status: 409, message: "You have already reported this listing" },
  });
  renderSection();

  expect(
    screen.getByText("You've already reported this listing. It's on record."),
  ).toBeInTheDocument();
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});
