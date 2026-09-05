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
  // position is the stored order, so the first image is the cover.
  const cover = listing.images[0];

  return (
    <Link
      to={`/listings/${listing.id}`}
      className="focus-visible:ring-accent block rounded-lg focus:outline-none focus-visible:ring-2"
    >
      <article className="border-line bg-surface hover:border-accent overflow-hidden rounded-lg border transition-colors">
        {/* An aspect box, not a bare <img>: a file has no size until it loads,
            so without a reserved shape every card's text is laid out against
            nothing and then shoved down when the bytes arrive. */}
        <div className="bg-surface-muted aspect-[4/3] w-full">
          {cover ? (
            // alt="": the title is directly below and is this link's accessible
            // name already, so describing the photo repeats it.
            <img src={cover.url} alt="" loading="lazy" className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full w-full items-center justify-center">
              {/* fill-current beats the mark's own fill attribute - a CSS rule
                  outranks a presentation attribute - so the logo follows the
                  theme here instead of staying brand green on a dark ground. */}
              {/* muted at half strength, measured: surface-soft came out at
                  1.27:1 on this ground in light and 1.23:1 in dark - an empty
                  grey box rather than a logo. This reads as a watermark. */}
              <svg className="text-muted h-10 w-10 fill-current opacity-50" aria-hidden="true">
                <use href="/icons.svg#brand-mark" />
              </svg>
            </div>
          )}
        </div>

        <div className="p-4">
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
        </div>
      </article>
    </Link>
  );
}
