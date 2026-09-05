import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import Profile from "../pages/Profile";
import { ModalProvider } from "../providers/ModalProvider";
import { ModalRoot } from "../components/modal/ModalRoot";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { useDeleteAvatar, useOwnProfile, useUploadAvatar } from "../api/profile";
import { useBlocks, useUnblock } from "../api/blocks";
import type { OwnProfile, User } from "../api/types";

vi.mock("../api/blocks", () => ({
  useBlocks: vi.fn(),
  useUnblock: vi.fn(),
}));

vi.mock("../api/profile", () => ({
  useOwnProfile: vi.fn(),
  useUpdateOwnProfile: vi.fn(),
  useChangePassword: vi.fn(),
  useUploadAvatar: vi.fn(),
  useDeleteAvatar: vi.fn(),
}));

const mockedProfile = vi.mocked(useOwnProfile);

const PROFILE: OwnProfile = {
  id: "u1",
  username: "or99",
  email: "oscarrogers@example.com",
  firstname: "Oscar",
  lastname: "Rogers",
  bio: null,
  phone_number: null,
  date_of_birth: null,
  location: "Espoo",
  avatar_url: null,
};

// A password-capable account, so the password section renders by default.
const USER: User = {
  id: "u1",
  username: "or99",
  email: "oscarrogers@example.com",
  role: "USER",
  has_password: true,
  providers: [],
};

// Profile itself only needs the modal system (delete-account, login prompt).
// The auth stub exists because the login modal it can open needs a context too.
const AUTH_STUB: AuthContextValue = {
  user: USER,
  isLoading: false,
  login: vi.fn().mockResolvedValue(undefined),
  signup: vi.fn(),
  logout: vi.fn().mockResolvedValue(undefined),
  restoreSession: vi.fn(),
};

const uploadAvatar = vi.fn();
const deleteAvatar = vi.fn();

beforeEach(() => {
  vi.mocked(useBlocks).mockReturnValue({
    data: [],
    isPending: false,
    isError: false,
  } as unknown as ReturnType<typeof useBlocks>);
  // Matches what BlockedUsersSection actually reads. The old mutateAsync
  // fixture passed only because an empty list short-circuits before the rows.
  vi.mocked(useUnblock).mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
    variables: undefined,
  } as unknown as ReturnType<typeof useUnblock>);
  uploadAvatar.mockReset().mockResolvedValue({ avatar_url: "/uploads/new.png" });
  deleteAvatar.mockReset().mockResolvedValue(undefined);
  vi.mocked(useUploadAvatar).mockReturnValue({
    mutateAsync: uploadAvatar,
    isPending: false,
  } as unknown as ReturnType<typeof useUploadAvatar>);
  vi.mocked(useDeleteAvatar).mockReturnValue({
    mutateAsync: deleteAvatar,
    isPending: false,
  } as unknown as ReturnType<typeof useDeleteAvatar>);
});

// Only the few fields the page actually reads; the real query result type
// is richer than any stub needs.
function renderPage(
  query: { data?: OwnProfile; isLoading?: boolean; error?: unknown },
  userOverride: User | null = AUTH_STUB.user,
  authLoading = false,
) {
  mockedProfile.mockReturnValue(query as ReturnType<typeof useOwnProfile>);
  // The blocked list links to each person, so the page needs a router as soon
  // as that list has rows - an empty fixture hides the requirement.
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <AuthContext.Provider value={{ ...AUTH_STUB, user: userOverride, isLoading: authLoading }}>
        <ModalProvider>
          <MemoryRouter>
            <Profile />
          </MemoryRouter>
          <ModalRoot />
        </ModalProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  );
}

test("shows a loading note while the profile is being fetched", () => {
  renderPage({ data: undefined, isLoading: true, error: null });
  expect(screen.getByText("Loading…")).toBeInTheDocument();
});

test("greets a signed-out visitor with a way to log in", async () => {
  const user = userEvent.setup();
  renderPage({
    data: undefined,
    isLoading: false,
    error: { status: 401, message: "authentication required" },
  });
  expect(screen.getByText(/You're signed out/, { exact: false })).toBeInTheDocument();

  await user.click(screen.getByRole("button", { name: "Log In" }));
  expect(screen.getByRole("heading", { name: "Log in" })).toBeInTheDocument();
});

test("renders the signed-in identity from the backend", () => {
  renderPage({ data: PROFILE, isLoading: false, error: null });

  expect(screen.getByText("or99")).toBeInTheDocument();
  expect(screen.getByText("oscarrogers@example.com")).toBeInTheDocument();
  // One initial from the username, same rule as the header avatar.
  expect(screen.getByText("O")).toBeInTheDocument();
  // A password-capable account gets the password section.
  expect(screen.getByText("Password")).toBeInTheDocument();
});

test("hides the password section for a provider-only (OAuth) account", () => {
  renderPage(
    { data: PROFILE, isLoading: false, error: null },
    { ...USER, has_password: false, providers: ["google"] },
  );

  expect(screen.getByText("or99")).toBeInTheDocument();
  // Neither the subheader nor the edit button of the password section appear.
  expect(screen.queryByText("Password")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Edit Password" })).not.toBeInTheDocument();
  // The rest of the page is unaffected.
  expect(screen.getByText("Contact Details")).toBeInTheDocument();
});

test("other failures surface their message rather than spinning forever", () => {
  renderPage({
    data: undefined,
    isLoading: false,
    error: { status: 500, message: "Something went wrong" },
  });
  expect(screen.getByText("Something went wrong")).toBeInTheDocument();
});

const WITH_AVATAR: OwnProfile = { ...PROFILE, avatar_url: "/uploads/stored.png" };

// alt="" on purpose - the picture is decorative beside the username - so it
// has no img role to query by.
test("renders the stored avatar rather than the initials fallback", () => {
  const { container } = renderPage({ data: WITH_AVATAR });
  expect(container.querySelector("img")).toHaveAttribute("src", "/uploads/stored.png");
});

// The whole path: the avatar button opens the picker, the picked file reaches
// the upload. Nothing below the modal is stubbed, so a broken onImageSelected
// wiring fails here rather than in review.
test("uploads the picture the user picks", async () => {
  const user = userEvent.setup();
  renderPage({ data: PROFILE });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));

  const file = new File(["x"], "me.png", { type: "image/png" });
  // document, not container: the modal renders through a portal. And fireEvent,
  // because the dropzone's input is display:none and userEvent won't click it.
  const input = document.querySelector('input[type="file"]');
  fireEvent.change(input as HTMLInputElement, { target: { files: [file] } });

  await user.click(await screen.findByRole("button", { name: "Save" }));

  expect(uploadAvatar).toHaveBeenCalledWith(file);
});

// Removal lives in the picture editor now, next to the thing it acts on, and
// the page decides whether to offer it by handing the modal an onRemove at all.
test("offers removal only once a picture is stored", async () => {
  const user = userEvent.setup();
  renderPage({ data: PROFILE });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  expect(screen.queryByRole("button", { name: "Remove photo" })).not.toBeInTheDocument();

  cleanup();
  renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  expect(screen.getByRole("button", { name: "Remove photo" })).toBeInTheDocument();
});

test("removing a picture calls the delete endpoint and closes the editor", async () => {
  const user = userEvent.setup();
  renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  await user.click(screen.getByRole("button", { name: "Remove photo" }));

  expect(deleteAvatar).toHaveBeenCalled();
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "Remove photo" })).not.toBeInTheDocument(),
  );
});

// The backend's own message is more useful than anything written here - 413
// says the image is too large, 415 says the type is wrong.
test("a rejected upload surfaces the reason and keeps no preview", async () => {
  uploadAvatar.mockRejectedValue({ status: 413, message: "Image is too large" });
  const user = userEvent.setup();
  const { container } = renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  const input = document.querySelector('input[type="file"]');
  fireEvent.change(input as HTMLInputElement, {
    target: { files: [new File(["x"], "big.png", { type: "image/png" })] },
  });
  await user.click(await screen.findByRole("button", { name: "Save" }));

  expect(await screen.findByRole("alert")).toHaveTextContent("Image is too large");
  // The picture was never stored, so the screen must fall back to what is.
  await waitFor(() => {
    expect(container.querySelector("img")).toHaveAttribute("src", "/uploads/stored.png");
  });
});

// The editor must be gone before the request settles, or the page's own
// "Removing…" line renders behind it and the button stays live for a second
// click. The stubbed isPending in beforeEach cannot show this - the mutation
// has to be caught mid-flight.
test("closes the editor before the delete answers, so the page can show progress", async () => {
  let finishDelete!: () => void;
  deleteAvatar.mockReturnValue(
    new Promise<void>((resolve) => {
      finishDelete = resolve;
    }),
  );
  vi.mocked(useDeleteAvatar).mockReturnValue({
    mutateAsync: deleteAvatar,
    isPending: true,
  } as unknown as ReturnType<typeof useDeleteAvatar>);

  const user = userEvent.setup();
  renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  await user.click(screen.getByRole("button", { name: "Remove photo" }));

  // This one catches the bug. The line below cannot: isPending is stubbed from
  // render, so "Removing…" is on the page before the click, and getByText does
  // not know a modal is covering it - it guards Profile's indicator existing at
  // all, not the closing.
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(screen.getByText("Removing…")).toBeInTheDocument();

  finishDelete();
});

// The modal is gone by the time the request answers, so the page is where a
// failure has to land - same as a rejected upload.
test("a rejected removal surfaces the reason", async () => {
  deleteAvatar.mockRejectedValue({ status: 500, message: "Something broke" });
  const user = userEvent.setup();
  renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  await user.click(screen.getByRole("button", { name: "Remove photo" }));

  expect(await screen.findByRole("alert")).toHaveTextContent("Something broke");
});

// The preview is a blob: URL that exists only in this tab. Leaving it in place
// after the upload looks correct on this screen and nowhere else - including
// after a reload - so the stored URL has to take over.
test("the stored picture takes over from the preview once the upload lands", async () => {
  const user = userEvent.setup();
  const { container } = renderPage({ data: WITH_AVATAR });

  await user.click(screen.getByRole("button", { name: "Edit profile picture" }));
  const input = document.querySelector('input[type="file"]');
  fireEvent.change(input as HTMLInputElement, {
    target: { files: [new File(["x"], "me.png", { type: "image/png" })] },
  });
  await user.click(await screen.findByRole("button", { name: "Save" }));

  await waitFor(() => {
    expect(container.querySelector("img")).toHaveAttribute("src", "/uploads/stored.png");
  });
});

// has_password decides whether the password section renders, and AuthProvider
// reports user as null until the session is restored - so rendering early
// tells a password account it has none, then corrects itself.
test("waits out the session restore before deciding about the password section", () => {
  renderPage({ data: PROFILE, isLoading: false }, null, true);

  expect(screen.getByText("Loading…")).toBeInTheDocument();
  expect(screen.queryByText("Password")).not.toBeInTheDocument();
});
