import { render, screen } from "@testing-library/react";

import OAuthButtons from "../components/forms/OAuthButtons";
import { useOAuthProviders } from "../api/oauth";

vi.mock("../api/oauth", () => ({ useOAuthProviders: vi.fn() }));

function renderButtons(data: string[] | undefined) {
  vi.mocked(useOAuthProviders).mockReturnValue({ data } as ReturnType<typeof useOAuthProviders>);
  render(<OAuthButtons divider="or sign in with email" />);
}

// The backend only registers a provider whose credentials are set, so a button
// it does not list is one that answers 404 - and because these are top-level
// navigations, clicking it leaves the app for a JSON error page.
test("offers only the providers the backend lists", () => {
  renderButtons(["google"]);

  expect(screen.getByRole("link", { name: "Continue with Google" })).toHaveAttribute(
    "href",
    "/api/v1/auth/oauth/google",
  );
  expect(screen.queryByRole("link", { name: "Continue with GitHub" })).not.toBeInTheDocument();
});

test("offers both when both are configured", () => {
  renderButtons(["github", "google"]);

  expect(screen.getByRole("link", { name: "Continue with Google" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Continue with GitHub" })).toBeInTheDocument();
});

// Display order is ours, not the response's: the backend sorts alphabetically,
// which would otherwise put GitHub first.
test("keeps its own order regardless of how the list arrives", () => {
  renderButtons(["github", "google"]);

  const names = screen.getAllByRole("link").map((link) => link.textContent);
  expect(names).toEqual(["Continue with Google", "Continue with GitHub"]);
});

// The divider introduces these buttons, so it cannot outlive them - a caller
// holding its own would be left saying "or" above a bare email form.
test("renders nothing at all, divider included, when none are configured", () => {
  renderButtons([]);

  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.queryByText("or sign in with email")).not.toBeInTheDocument();
});

// Fails closed: a provider we cannot confirm is one that very likely would not
// have worked either. Covers both the in-flight and the failed request.
test("renders nothing while the list is unknown", () => {
  renderButtons(undefined);

  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.queryByText("or sign in with email")).not.toBeInTheDocument();
});

test("shows the divider once there is something to divide", () => {
  renderButtons(["google"]);

  expect(screen.getByText("or sign in with email")).toBeInTheDocument();
});
