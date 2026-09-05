import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { SellerRating } from "./SellerRating";
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
  showSeller?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <Link
      to={`/listings/${listing.id}`}
      className="focus-visible:ring-accent block rounded-lg focus:outline-none focus-visible:ring-2"
    >
      <article className="border-line bg-surface hover:border-accent rounded-lg border p-4 transition-colors">
        <h2 className="text-foreground text-item-title font-semibold">{listing.title}</h2>
        <p className="text-muted mt-1 text-sm">{categoryName}</p>
        <p className="text-accent mt-2 font-medium">
          €{Number(listing.price).toFixed(2)} / {listing.unit}
        </p>
        <p className="text-muted text-sm">
          {t("listings.available", { qty: listing.quantity, unit: listing.unit })}
        </p>

        {/* Plain text, not a link: the card is an anchor now, and an anchor
            inside an anchor fires both on one click. */}
        {showSeller && listing.seller && (
          <div className="border-line mt-3 flex items-center gap-2 border-t pt-3">
            <Avatar
              size="sm"
              initials={deriveInitials(listing.seller.username)}
              imageUrl={listing.seller.avatar_url ?? undefined}
            />
            <div className="min-w-0">
              <span className="text-muted block truncate text-sm">
                <span className="sr-only">{t("listings.seller")}: </span>
                {listing.seller.username}
              </span>
              <SellerRating rating={listing.seller.rating} />
            </div>
          </div>
        )}
      </article>
    </Link>
  );
}
