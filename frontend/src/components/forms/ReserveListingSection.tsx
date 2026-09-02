import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
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
  const { t } = useTranslation();
  const { openModal } = useModal();

  // Fetches by id rather than taking a listing prop: React Query dedupes by
  // key, so this costs nothing next to the page's own fetch and the section
  // drops into the #21 stub with one line.
  const { data: listing, isPending, isError, refetch } = useListing(listingId);

  // An empty id leaves the query disabled, which React Query still reports as
  // pending - so answer it before the skeleton branch spins forever.
  if (!listingId) return null;
  if (isPending) return <Skeleton className="h-40 w-full" />;

  // Without this the box just disappears, on a page that has nothing else on
  // it - the same silent blank Orders.tsx already avoids.
  if (isError || !listing) {
    return (
      <Skeleton
        variant="error"
        className="h-40 w-full"
        message={t("orders.reserve.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (user?.id === listing.seller_id) {
    return <p className="text-muted text-sm">{t("orders.reserve.ownListing")}</p>;
  }

  if (listing.quantity <= 0) {
    return <p className="text-muted text-sm">{t("orders.reserve.soldOut")}</p>;
  }

  if (!user) {
    return (
      <div className="border-line space-y-3 rounded-lg border p-4">
        <p className="text-muted text-sm">{t("orders.reserve.logInToReserve")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  return <ReserveForm listing={listing} />;
}

// Split out so the listing is a plain prop: TypeScript drops the "not
// undefined" narrowing inside the async handler, and a prop never loses it.
function ReserveForm({ listing }: { listing: Listing }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const createOrder = useCreateOrder();

  // Kept as text so the field can be cleared while typing; parsed below.
  const [quantityText, setQuantityText] = useState("1");
  const [error, setError] = useState<string | null>(null);

  const available = listing.quantity;
  // Number, not parseInt: parseInt("1.5") is 1, so the field would read 1.5
  // while we reserved 1. Number("1.5") keeps Number.isInteger meaningful.
  const quantity = Number(quantityText);
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
      setError(isApiError(err) ? err.message : t("orders.reserve.error"));
    }
  }

  return (
    <div className="border-line space-y-3 rounded-lg border p-4">
      <div>
        <p className="text-accent font-medium">
          €{listing.price.toFixed(2)} / {listing.unit}
        </p>
        <p className="text-muted text-sm">
          {t("listings.available", { qty: available, unit: listing.unit })}
        </p>
      </div>

      <div className="flex items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="reserve-quantity" className="text-muted text-sm">
            {t("orders.reserve.quantityLabel", { unit: listing.unit })}
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
          {createOrder.isPending ? t("orders.reserve.submitting") : t("orders.reserve.submit")}
        </Button>
      </div>

      {valid ? (
        <p className="text-muted text-sm">
          {t("orders.reserve.total", { total: (listing.price * quantity).toFixed(2) })}
        </p>
      ) : (
        <p className="text-muted text-sm">
          {t("orders.reserve.chooseBetween", { max: available, unit: listing.unit })}
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
