import { useTranslation } from "react-i18next";

import Avatar from "./Avatar";
import { deriveInitials } from "../../lib/initials";
import type { Listing } from "../../api/types";

// presentation only: takes one Listing and renders it, no data fetching here
export function ListingCard({
  listing,
  categoryName,
  showSeller = true,
}: {
  listing: Listing;
  categoryName: string;
  // Off on a seller's own profile, where every card would name the person
  // whose page it already is.
  showSeller?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <article className="border-line bg-surface rounded-lg border p-4">
      <h2 className="text-foreground font-semibold">{listing.title}</h2>
      <p className="text-muted mt-1 text-sm">{categoryName}</p>
      <p className="text-accent mt-2 font-medium">
        €{Number(listing.price).toFixed(2)} / {listing.unit}
      </p>
      <p className="text-muted text-sm">
        {t("listings.available", { qty: listing.quantity, unit: listing.unit })}
      </p>

      {showSeller && listing.seller && (
        <div className="border-line mt-3 flex items-center gap-2 border-t pt-3">
          <Avatar
            size="sm"
            initials={deriveInitials(listing.seller.username)}
            imageUrl={listing.seller.avatar_url ?? undefined}
          />
          <span className="text-muted truncate text-sm">
            <span className="sr-only">{t("listings.seller")}: </span>
            {listing.seller.username}
          </span>
        </div>
      )}
    </article>
  );
}
