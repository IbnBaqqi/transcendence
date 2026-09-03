import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { StartConversationSection } from "../components/forms/StartConversationSection";
import { useListing } from "../api/listings";
import { useStartConversation } from "../api/conversations";
import { AuthContext, type AuthContextValue } from "../providers/AuthContext";
import { ModalProvider } from "../providers/ModalProvider";
import { makeListing, makeConversation, BUYER_ID, SELLER_ID } from "../test/factories";
import type { Listing, User } from "../api/types";

vi.mock("../api/listings", () => ({ useListing: vi.fn() }));
vi.mock("../api/conversations", () => ({ useStartConversation: vi.fn() }));

const mutateAsync = vi.fn();

const VIEWER: User = {
  id: BUYER_ID,
  username: "buyer",
  email: "b@x.test",
  role: "user",
  has_password: true,
  providers: [],
};

beforeEach(() => {
  mutateAsync.mockReset().mockResolvedValue(makeConversation({ id: "c-new" }));
  vi.mocked(useStartConversation).mockReturnValue({
    mutateAsync,
    isPending: false,
  } as unknown as ReturnType<typeof useStartConversation>);
});

function renderSection(listing: Listing, user: User | null = VIEWER) {
  vi.mocked(useListing).mockReturnValue({
    data: listing,
    isPending: false,
  } as ReturnType<typeof useListing>);

  const auth: AuthContextValue = {
    user,
    isLoading: false,
    login: vi.fn(),
    signup: vi.fn(),
    logout: vi.fn(),
    restoreSession: vi.fn(),
  };

  return render(
    <AuthContext.Provider value={auth}>
      <ModalProvider>
        <StartConversationSection listingId={listing.id} />
      </ModalProvider>
    </AuthContext.Provider>,
  );
}

describe("StartConversationSection", () => {
  test("sends the first message with the listing id", async () => {
    const user = userEvent.setup();
    const listing = makeListing();
    renderSection(listing);

    await user.click(screen.getByRole("button", { name: "Message the seller" }));
    await user.type(screen.getByLabelText(/Your message/), "Still available?");
    await user.click(screen.getByRole("button", { name: "Send request" }));

    expect(mutateAsync).toHaveBeenCalledWith({
      listing_id: listing.id,
      body: "Still available?",
    });
  });

  test("won't send an empty message", async () => {
    const user = userEvent.setup();
    renderSection(makeListing());

    await user.click(screen.getByRole("button", { name: "Message the seller" }));

    expect(screen.getByRole("button", { name: "Send request" })).toBeDisabled();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  test("the seller can't message themselves", () => {
    renderSection(makeListing(), { ...VIEWER, id: SELLER_ID });
    expect(screen.queryByRole("button", { name: "Message the seller" })).not.toBeInTheDocument();
  });

  test("a signed-out visitor is offered the login", () => {
    renderSection(makeListing(), null);
    expect(
      screen.getByRole("button", { name: "Log in to message the seller" }),
    ).toBeInTheDocument();
  });

  // A duplicate 409 doesn't hand back the existing thread, so the only useful
  // thing to offer is the list.
  test("contacting twice points at the existing thread", async () => {
    mutateAsync.mockRejectedValue({
      status: 409,
      message: "You have already contacted this seller about this listing",
    });
    const user = userEvent.setup();
    renderSection(makeListing());

    await user.click(screen.getByRole("button", { name: "Message the seller" }));
    await user.type(screen.getByLabelText(/Your message/), "hello again");
    await user.click(screen.getByRole("button", { name: "Send request" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("already contacted");
    expect(screen.getByRole("button", { name: "Open your messages" })).toBeInTheDocument();
  });
});
