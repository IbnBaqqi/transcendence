import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DataExportSection } from "../components/forms/DataExportSection";
import { api } from "../api/client";

vi.mock("../api/client", () => ({ api: { get: vi.fn() } }));

const createObjectURL = vi.fn(() => "blob:fake");
const revokeObjectURL = vi.fn();

beforeEach(() => {
  Object.assign(URL, { createObjectURL, revokeObjectURL });
});

test("downloads the export as a file rather than rendering it", async () => {
  const user = userEvent.setup();
  vi.mocked(api.get).mockResolvedValue({ data: { exported_at: "2026-09-05T00:00:00Z" } });
  const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

  render(<DataExportSection />);
  await user.click(screen.getByRole("button", { name: "Download my data" }));

  await waitFor(() => expect(api.get).toHaveBeenCalledWith("/me/export"));
  expect(click).toHaveBeenCalled();
  // Nothing of the document reaches the page: it is handed to the browser.
  expect(screen.queryByText(/exported_at/)).not.toBeInTheDocument();
});

// An object URL pins the blob in memory until it is released, and this blob is
// the user's entire record.
test("releases the object URL after handing the file over", async () => {
  const user = userEvent.setup();
  vi.mocked(api.get).mockResolvedValue({ data: {} });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

  render(<DataExportSection />);
  await user.click(screen.getByRole("button", { name: "Download my data" }));

  await waitFor(() => expect(revokeObjectURL).toHaveBeenCalledWith("blob:fake"));
});

test("says so when the download could not be prepared", async () => {
  const user = userEvent.setup();
  vi.mocked(api.get).mockRejectedValue(new Error("nope"));

  render(<DataExportSection />);
  await user.click(screen.getByRole("button", { name: "Download my data" }));

  expect(await screen.findByRole("alert")).toHaveTextContent("Couldn't prepare the download.");
});
