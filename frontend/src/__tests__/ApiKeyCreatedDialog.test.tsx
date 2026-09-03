import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ApiKeyCreatedDialog } from "../components/modal/ApiKeyCreatedDialog";
import type { CreatedApiKey } from "../api/types";

const KEY: CreatedApiKey = {
  id: "k1",
  name: "ci pipeline",
  key_prefix: "fk_live_a3f9",
  key: "fk_live_a3f9c2e18b7d4f60",
  last_used_at: null,
  revoked_at: null,
  created_at: "1970-01-01T00:00:00Z",
};

// Must run AFTER userEvent.setup(), which installs a clipboard stub of its own
// and would otherwise replace this one.
function stubClipboard(impl: () => Promise<void>) {
  const writeText = vi.fn(impl);
  Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
  return writeText;
}

describe("ApiKeyCreatedDialog", () => {
  test("shows the key and says it will not be shown again", () => {
    render(<ApiKeyCreatedDialog apiKey={KEY} onClose={vi.fn()} />);

    expect(screen.getByLabelText("API key")).toHaveValue(KEY.key);
    expect(screen.getByText(/only time it will be shown/)).toBeInTheDocument();
  });

  test("copying puts the key on the clipboard and confirms it", async () => {
    const user = userEvent.setup();
    const writeText = stubClipboard(() => Promise.resolve());
    render(<ApiKeyCreatedDialog apiKey={KEY} onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(writeText).toHaveBeenCalledWith(KEY.key);
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  // The key is unrecoverable, so a failed copy must not be a lost key.
  test("a failed copy leaves the key on screen with a way out", async () => {
    const user = userEvent.setup();
    stubClipboard(() => Promise.reject(new Error("denied")));
    render(<ApiKeyCreatedDialog apiKey={KEY} onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("copy it manually");
    expect(screen.getByLabelText("API key")).toHaveValue(KEY.key);
  });
});
