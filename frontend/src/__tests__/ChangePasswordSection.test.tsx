import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ChangePasswordSection } from "../components/forms/ChangePasswordSection";
import { useChangePassword } from "../api/profile";

// The hook owns the network; replace it wholesale so tests control exactly
// what gets sent over the wire and capture the mapped payload.
vi.mock("../api/profile", () => ({
  useChangePassword: vi.fn(),
}));

const mockedChangePassword = vi.mocked(useChangePassword);

function changePasswordMutation(
  mutateAsync: ReturnType<typeof vi.fn> = vi.fn().mockResolvedValue(undefined),
) {
  return { mutateAsync } as unknown as ReturnType<typeof useChangePassword>;
}

beforeEach(() => {
  vi.clearAllMocks();
  mockedChangePassword.mockReturnValue(changePasswordMutation());
});

test("shows the password placeholder before editing", () => {
  render(<ChangePasswordSection />);

  expect(screen.getByText("********")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Edit Password" })).toBeInTheDocument();
});

test("reveals the password fields when entering edit mode", async () => {
  const user = userEvent.setup();
  render(<ChangePasswordSection />);

  await user.click(screen.getByRole("button", { name: "Edit Password" }));

  expect(screen.getByLabelText("Current password")).toBeInTheDocument();
  expect(screen.getByLabelText("New password")).toBeInTheDocument();
  expect(screen.getByLabelText("Confirm password")).toBeInTheDocument();
});

test("saves mapped values and leaves edit mode", async () => {
  const mutateAsync = vi.fn().mockResolvedValue(undefined);
  mockedChangePassword.mockReturnValue(changePasswordMutation(mutateAsync));
  const user = userEvent.setup();
  render(<ChangePasswordSection />);

  await user.click(screen.getByRole("button", { name: "Edit Password" }));
  await user.type(screen.getByLabelText("Current password"), "oldSecret123");
  await user.type(screen.getByLabelText("New password"), "newSecret123");
  await user.type(screen.getByLabelText("Confirm password"), "newSecret123");
  await user.click(screen.getByRole("button", { name: "Save" }));

  // camelCase form fields map to snake_case, and confirmPassword never leaves
  // the client.
  await waitFor(() =>
    expect(mutateAsync).toHaveBeenCalledWith({
      current_password: "oldSecret123",
      new_password: "newSecret123",
    }),
  );
  expect(await screen.findByRole("button", { name: "Edit Password" })).toBeInTheDocument();
});

test("a server error shows inline and keeps edit mode open", async () => {
  mockedChangePassword.mockReturnValue(
    changePasswordMutation(
      vi.fn().mockRejectedValue({ status: 403, message: "Current password is incorrect" }),
    ),
  );
  const user = userEvent.setup();
  render(<ChangePasswordSection />);

  await user.click(screen.getByRole("button", { name: "Edit Password" }));
  await user.type(screen.getByLabelText("Current password"), "wrongSecret123");
  await user.type(screen.getByLabelText("New password"), "newSecret123");
  await user.type(screen.getByLabelText("Confirm password"), "newSecret123");
  await user.click(screen.getByRole("button", { name: "Save" }));

  expect(await screen.findByRole("alert")).toHaveTextContent("Current password is incorrect");
  expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
});

test("cancel discards edits instead of saving them", async () => {
  const mutateAsync = vi.fn().mockResolvedValue(undefined);
  mockedChangePassword.mockReturnValue(changePasswordMutation(mutateAsync));
  const user = userEvent.setup();
  render(<ChangePasswordSection />);

  await user.click(screen.getByRole("button", { name: "Edit Password" }));
  await user.type(screen.getByLabelText("New password"), "newSecret123");
  await user.type(screen.getByLabelText("Confirm password"), "newSecret123");
  await user.click(screen.getByRole("button", { name: "Cancel" }));

  expect(mutateAsync).not.toHaveBeenCalled();
  expect(screen.getByText("********")).toBeInTheDocument();
});

test("blocks submit while new and confirm password mismatch", async () => {
  const mutateAsync = vi.fn().mockResolvedValue(undefined);
  mockedChangePassword.mockReturnValue(changePasswordMutation(mutateAsync));
  const user = userEvent.setup();
  render(<ChangePasswordSection />);

  await user.click(screen.getByRole("button", { name: "Edit Password" }));
  await user.type(screen.getByLabelText("Current password"), "oldSecret123");
  await user.type(screen.getByLabelText("New password"), "newSecret123");
  await user.type(screen.getByLabelText("Confirm password"), "differentSecret456");
  await user.click(screen.getByRole("button", { name: "Save" }));

  expect(mutateAsync).not.toHaveBeenCalled();
});
