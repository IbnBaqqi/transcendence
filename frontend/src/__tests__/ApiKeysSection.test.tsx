import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ApiKeysSection } from "../components/forms/ApiKeysSection";
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from "../api/apiKeys";
import type { ApiKey, CreatedApiKey } from "../api/types";

vi.mock("../api/apiKeys", () => ({
  useApiKeys: vi.fn(),
  useCreateApiKey: vi.fn(),
  useRevokeApiKey: vi.fn(),
}));

const calls = { create: vi.fn(), reset: vi.fn(), revoke: vi.fn() };

const mutation = <T,>(fn: unknown, reset = vi.fn()) =>
  ({ mutateAsync: fn, isPending: false, reset }) as unknown as T;

function makeKey(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: "k1",
    name: "ci pipeline",
    key_prefix: "fk_live_a3f9",
    last_used_at: null,
    revoked_at: null,
    created_at: "2026-09-01T10:00:00Z",
    ...overrides,
  };
}

const CREATED: CreatedApiKey = { ...makeKey(), key: "fk_live_a3f9c2e18b7d4f60" };

beforeEach(() => {
  calls.create.mockReset().mockResolvedValue(CREATED);
  calls.reset.mockReset();
  calls.revoke.mockReset().mockResolvedValue(undefined);
  vi.mocked(useCreateApiKey).mockReturnValue(mutation(calls.create, calls.reset));
  vi.mocked(useRevokeApiKey).mockReturnValue(mutation(calls.revoke));
});

function renderSection(keys: ApiKey[] = [], state = {}) {
  vi.mocked(useApiKeys).mockReturnValue({
    data: keys,
    isPending: false,
    isError: false,
    ...state,
  } as unknown as ReturnType<typeof useApiKeys>);
  return render(<ApiKeysSection />);
}

describe("ApiKeysSection", () => {
  // Closing must drop BOTH copies of the secret: our own state, and the one
  // TanStack keeps on mutation.data until reset().
  test("creates a key with the trimmed name, shows it once, and drops it on close", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText("Name"), "  ci pipeline  ");
    await user.click(screen.getByRole("button", { name: "Create key" }));

    expect(calls.create).toHaveBeenCalledWith("ci pipeline");
    expect(await screen.findByLabelText("API key")).toHaveValue(CREATED.key);

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(screen.queryByLabelText("API key")).not.toBeInTheDocument();
    expect(calls.reset).toHaveBeenCalled();
  });

  test("won't create a key with no name", () => {
    renderSection();
    expect(screen.getByRole("button", { name: "Create key" })).toBeDisabled();
  });

  test("shows the prefix and that a key has never been used", () => {
    renderSection([makeKey()]);
    expect(screen.getByText(/fk_live_a3f9/)).toBeInTheDocument();
    expect(screen.getByText(/never used/)).toBeInTheDocument();
  });

  // Revoked keys stay listed so a user can see what they switched off.
  test("a revoked key is shown as revoked, not hidden, and offers no revoke", () => {
    renderSection([makeKey({ revoked_at: "2026-09-02T10:00:00Z" })]);

    expect(screen.getByText("ci pipeline")).toBeInTheDocument();
    expect(screen.getByText("Revoked")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Revoke" })).not.toBeInTheDocument();
  });

  test("revoking takes two clicks", async () => {
    const user = userEvent.setup();
    renderSection([makeKey()]);

    await user.click(screen.getByRole("button", { name: "Revoke" }));
    expect(calls.revoke).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Confirm revoke" }));
    expect(calls.revoke).toHaveBeenCalledWith("k1");
  });

  // RevokeKey matches revoked_at IS NULL, so a second revoke answers 404 for a
  // row that is plainly on screen.
  test("a 404 on revoke reads as already revoked, not as missing", async () => {
    calls.revoke.mockRejectedValue({ status: 404, message: "API key not found" });
    const user = userEvent.setup();
    renderSection([makeKey()]);

    await user.click(screen.getByRole("button", { name: "Revoke" }));
    await user.click(screen.getByRole("button", { name: "Confirm revoke" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("already revoked");
  });
});
