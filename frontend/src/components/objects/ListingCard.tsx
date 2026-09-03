import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import type { Listing } from "../../api/types";

// presentation only: takes one Listing and renders it, no data fetching here
export function ListingCard({ listing, categoryName }: { listing: Listing; categoryName: string }) {
  const { t } = useTranslation();
  return (
    <Link
      to={`/listings/${listing.id}`}
      className="focus-visible:ring-accent block rounded-lg focus:outline-none focus-visible:ring-2"
    >
      <article className="border-line bg-surface hover:border-accent rounded-lg border p-4 transition-colors">
        <h2 className="text-foreground font-semibold">{listing.title}</h2>
        <p className="text-muted mt-1 text-sm">{categoryName}</p>
        <p className="text-accent mt-2 font-medium">
          €{Number(listing.price).toFixed(2)} / {listing.unit}
        </p>
        <p className="text-muted text-sm">
          {t("listings.available", { qty: listing.quantity, unit: listing.unit })}
        </p>
      </article>
    </Link>
  );
}
