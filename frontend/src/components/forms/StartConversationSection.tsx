import { useState } from "react";

import { useListing } from "../../api/listings";
import { useStartConversation } from "../../api/conversations";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { useModal } from "../../providers/modalContext";
import Button from "../objects/Button";
import type { Listing } from "../../api/types";

export function StartConversationSection({ listingId }: { listingId: string }) {
  const { user } = useAuth();
  const { openModal } = useModal();

  // Same shape as ReserveListingSection: fetches by id so it drops into the
  // #21 stub with one line, and React Query dedupes the two.
  const { data: listing, isPending } = useListing(listingId);

  if (!listingId || isPending || !listing) return null;
  if (user?.id === listing.seller_id) return null;

  if (!user) {
    return (
      <Button variant="secondary" onClick={() => openModal("login")}>
        Log in to message the seller
      </Button>
    );
  }

  return <StartConversationForm listing={listing} />;
}

// Split out for the same reason as ReserveForm: TypeScript loses the "not
// undefined" narrowing inside an async handler, a prop never does.
function StartConversationForm({ listing }: { listing: Listing }) {
  const { openChat } = useModal();
  const start = useStartConversation();

  const [open, setOpen] = useState(false);
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);
  // A duplicate is not really a failure - there is a thread already, we just
  // aren't told which one, so the only useful move is to open the list.
  const [duplicate, setDuplicate] = useState(false);

  async function handleSend() {
    setError(null);
    setDuplicate(false);
    try {
      const conversation = await start.mutateAsync({
        listing_id: listing.id,
        body: body.trim(),
      });
      setOpen(false);
      setBody("");
      openChat(conversation.id);
    } catch (err) {
      if (isApiError(err) && err.status === 409) {
        setDuplicate(true);
        setError(err.message);
        return;
      }
      setError(isApiError(err) ? err.message : "Couldn't start that conversation.");
    }
  }

  if (!open) {
    return (
      <Button variant="secondary" onClick={() => setOpen(true)}>
        Message the seller
      </Button>
    );
  }

  return (
    <div className="border-line space-y-3 rounded-lg border p-4">
      <label htmlFor="first-message" className="text-muted block text-sm">
        Your message about {listing.title}
      </label>
      <textarea
        id="first-message"
        rows={3}
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="Is this still available?"
        className="border-line bg-surface text-foreground w-full rounded-md border px-3 py-2"
      />

      <div className="flex gap-2">
        <Button
          variant="primary"
          disabled={!body.trim() || start.isPending}
          onClick={() => void handleSend()}
        >
          {start.isPending ? "Sending…" : "Send request"}
        </Button>
        <Button variant="secondary" onClick={() => setOpen(false)}>
          Cancel
        </Button>
      </div>

      <p className="text-muted text-sm">
        The seller has to accept before you can exchange more messages.
      </p>

      {error && (
        <p role="alert" className="text-berry-500 text-sm">
          {error}
        </p>
      )}
      {duplicate && (
        <Button variant="tertiary" onClick={() => openChat()}>
          Open your messages
        </Button>
      )}
    </div>
  );
}
