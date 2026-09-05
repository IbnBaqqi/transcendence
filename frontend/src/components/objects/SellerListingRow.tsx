import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useDeleteListing } from "../../api/listings";
import { isApiError } from "../../api/client";
import { hasSaleInProgress } from "../../lib/sellerStats";
import Button from "./Button";
import type { Listing, Order } from "../../api/types";
import type { ListingStats } from "../../lib/sellerStats";

export function SellerListingRow({
  listing,
  stats,
  orders,
}: {
  listing: Listing;
  stats: ListingStats;
  orders: Order[];
}) {
  const { t } = useTranslation();

  return (
    // An <article>, not a <Link> around everything: the delete button lives in
    // here now, and interactive content cannot nest inside an anchor.
    <article className="border-line bg-surface rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-foreground text-item-title font-medium">
          <Link
            to={`/listings/${listing.id}`}
            className="focus-visible:ring-accent rounded hover:underline focus:outline-none focus-visible:ring-2"
          >
            {listing.title}
          </Link>
        </h3>
        <span className="text-accent shrink-0 font-medium">
          €{listing.price.toFixed(2)} / {listing.unit}
        </span>
      </div>

      <p className="text-muted mt-1 text-sm">
        {listing.quantity === 0
          ? t("pages.dashboard.listing.soldOut")
          : stats.sold === 0
            ? // "4 of 4 left" is noise before anything has sold.
              t("pages.dashboard.listing.allRemaining", {
                qty: listing.quantity,
                unit: listing.unit,
              })
            : t("pages.dashboard.listing.remaining", {
                left: listing.quantity,
                original: stats.original,
                unit: listing.unit,
              })}
        {stats.sold > 0 && <> · {t("pages.dashboard.listing.sold", { count: stats.sold })}</>}
      </p>

      <DeleteListing listing={listing} inFlight={hasSaleInProgress(listing, orders)} />
    </article>
  );
}

function DeleteListing({ listing, inFlight }: { listing: Listing; inFlight: boolean }) {
  const { t } = useTranslation();
  const remove = useDeleteListing();
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);

  // The button the keyboard was standing on is replaced by two others, so
  // without this focus falls to the body and the confirm step is unreachable
  // without tabbing back in from the top.
  useEffect(() => {
    if (confirming) confirmRef.current?.focus();
  }, [confirming]);

  async function handleDelete() {
    setError(null);
    try {
      await remove.mutateAsync(listing.id);
    } catch (err) {
      // The row is still on screen, so the reason belongs beside it. A sale can
      // start between the render and the click, and the server has the final
      // say - this is where that answer lands.
      setError(isApiError(err) ? err.message : t("pages.dashboard.listing.deleteFailed"));
      setConfirming(false);
    }
  }

  // Above the error on purpose: a refetch landing after a failed delete
  // replaces the server's 409 with this, which is the same fact in better copy
  // and takes the button away with it.
  if (inFlight) {
    return <p className="text-muted mt-3 text-sm">{t("pages.dashboard.listing.saleInProgress")}</p>;
  }

  return (
    <div className="mt-3 space-y-2">
      {confirming ? (
        <>
          <p className="text-muted text-sm">{t("pages.dashboard.listing.deleteWarning")}</p>
          <div className="flex flex-wrap gap-2">
            <Button
              ref={confirmRef}
              variant="primary"
              disabled={remove.isPending}
              onClick={() => void handleDelete()}
            >
              {remove.isPending
                ? t("pages.dashboard.listing.deleting")
                : t("pages.dashboard.listing.confirmDelete")}
            </Button>
            <Button variant="secondary" onClick={() => setConfirming(false)}>
              {t("common.cancel")}
            </Button>
          </div>
        </>
      ) : (
        <Button variant="secondary" onClick={() => setConfirming(true)}>
          {t("pages.dashboard.listing.delete")}
        </Button>
      )}

      {error && (
        <p role="alert" className="text-danger text-sm">
          {error}
        </p>
      )}
    </div>
  );
}
