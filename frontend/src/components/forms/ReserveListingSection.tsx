import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";

import { useListing } from "../../api/listings";
import { useCreateOrder } from "../../api/orders";
import { isApiError } from "../../api/client";
import { keys } from "../../api/queryKeys";
import { useAuth } from "../../hooks/useAuth";
import { useModal } from "../../providers/modalContext";
import Button from "../objects/Button";
import { Skeleton } from "../objects/Skeleton";
import type { Listing } from "../../api/types";

export function ReserveListingSection({ listingId }: { listingId: string }) {
  const { user } = useAuth();
  const { openModal } = useModal();

  // Fetches by id rather than taking a listing prop: React Query dedupes by
  // key, so this costs nothing next to the page's own fetch and the section
  // drops into the #21 stub with one line.
  const { data: listing, isPending } = useListing(listingId);

  // An empty id leaves the query disabled, which React Query still reports as
  // pending - so answer it before the skeleton branch spins forever.
  if (!listingId) return null;
  if (isPending) return <Skeleton className="h-40 w-full" />;
  if (!listing) return null;

  if (user?.id === listing.seller_id) {
    return <p className="text-muted text-sm">This is your own listing.</p>;
  }

  if (listing.quantity <= 0) {
    return <p className="text-muted text-sm">Sold out - nothing left to reserve.</p>;
  }

  if (!user) {
    return (
      <div className="border-line space-y-3 rounded-lg border p-4">
        <p className="text-muted text-sm">Log in to reserve from this seller.</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          Log In
        </Button>
      </div>
    );
  }

  return <ReserveForm listing={listing} />;
}

// Split out so the listing is a plain prop: TypeScript drops the "not
// undefined" narrowing inside the async handler, and a prop never loses it.
function ReserveForm({ listing }: { listing: Listing }) {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const createOrder = useCreateOrder();

  // Kept as text so the field can be cleared while typing; parsed below.
  const [quantityText, setQuantityText] = useState("1");
  const [error, setError] = useState<string | null>(null);

  const available = listing.quantity;
  const quantity = Number.parseInt(quantityText, 10);
  const valid = Number.isInteger(quantity) && quantity >= 1 && quantity <= available;

  async function handleReserve() {
    setError(null);
    try {
      const order = await createOrder.mutateAsync({ listing_id: listing.id, quantity });
      navigate(`/orders/${order.id}`);
    } catch (err) {
      // 409 is the sold-out race: someone took the rest while this page sat
      // open. Refetch so the max drops to what's actually left.
      if (isApiError(err) && err.status === 409) {
        void qc.invalidateQueries({ queryKey: keys.listings.all });
        setError(err.message);
        return;
      }
      setError(isApiError(err) ? err.message : "Couldn't reserve this listing. Please try again.");
    }
  }

  return (
    <div className="border-line space-y-3 rounded-lg border p-4">
      <div>
        <p className="text-accent font-medium">
          €{listing.price.toFixed(2)} / {listing.unit}
        </p>
        <p className="text-muted text-sm">
          {available} {listing.unit} available
        </p>
      </div>

      <div className="flex items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="reserve-quantity" className="text-muted text-sm">
            Quantity ({listing.unit})
          </label>
          <input
            id="reserve-quantity"
            type="number"
            min={1}
            max={available}
            value={quantityText}
            onChange={(e) => setQuantityText(e.target.value)}
            className="border-line bg-surface text-foreground w-24 rounded-md border px-3 py-2"
          />
        </div>
        <Button
          variant="primary"
          disabled={!valid || createOrder.isPending}
          onClick={() => void handleReserve()}
        >
          {createOrder.isPending ? "Reserving…" : "Request to buy"}
        </Button>
      </div>

      {valid ? (
        <p className="text-muted text-sm">
          Total €{(listing.price * quantity).toFixed(2)} - you pay the seller directly, not
          Metsätori.
        </p>
      ) : (
        <p className="text-muted text-sm">
          Choose between 1 and {available} {listing.unit}.
        </p>
      )}

      {error && (
        <p role="alert" className="text-berry-500 text-sm">
          {error}
        </p>
      )}
    </div>
  );
}
