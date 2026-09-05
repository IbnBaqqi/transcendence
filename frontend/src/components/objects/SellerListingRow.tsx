import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import type { Listing } from "../../api/types";
import type { ListingStats } from "../../lib/sellerStats";

export function SellerListingRow({ listing, stats }: { listing: Listing; stats: ListingStats }) {
  const { t } = useTranslation();

  return (
    <Link
      to={`/listings/${listing.id}`}
      className="focus-visible:ring-accent block rounded-lg focus:outline-none focus-visible:ring-2"
    >
      <article className="border-line bg-surface hover:border-accent rounded-lg border p-4 transition-colors">
        <div className="flex items-start justify-between gap-3">
          <h3 className="text-foreground text-item-title font-medium">{listing.title}</h3>
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
      </article>
    </Link>
  );
}
