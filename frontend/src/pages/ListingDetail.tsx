import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { ReserveListingSection } from "../components/forms/ReserveListingSection";
import { StartConversationSection } from "../components/forms/StartConversationSection";
import { ReportListingSection } from "../components/forms/ReportListingSection";
import { SellerFollowSection } from "../components/objects/SellerFollowSection";
import { Skeleton } from "../components/objects/Skeleton";
import { useListing } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { isApiError } from "../api/client";
import type { Listing } from "../api/types";

export default function ListingDetail() {
  const { id = "" } = useParams();
  const { t } = useTranslation();
  // The four sections below each ask for this listing too, and React Query
  // keys the cache by id - so the page reading it costs no extra request.
  const { data: listing, isPending, isError, error, refetch } = useListing(id);
  const categoryName = useLocalizedCategoryNames();

  return (
    <div className="max-w-page mx-auto px-4 py-8">
      {isPending ? (
        <Skeleton className="h-72 w-full" />
      ) : isError || !listing ? (
        <Skeleton
          variant="error"
          className="h-40 w-full"
          message={isApiError(error) ? error.message : t("pages.listingDetail.loadError")}
          // A deleted listing does not come back, so offering the button only
          // buys another 404.
          onRetry={isApiError(error) && error.status === 404 ? undefined : () => refetch()}
        />
      ) : (
        <>
          <Gallery listing={listing} />

          <h1 className="text-foreground text-page-title mt-6 font-bold">{listing.title}</h1>
          <p className="text-muted mt-1 text-sm">{categoryName(listing.category)}</p>
          <p className="text-accent text-section mt-3 font-medium">
            €{Number(listing.price).toFixed(2)} / {listing.unit}
          </p>
          <p className="text-muted text-sm">
            {t("listings.available", { qty: listing.quantity, unit: listing.unit })}
          </p>
          {listing.description && (
            <p className="text-foreground mt-4 whitespace-pre-line">{listing.description}</p>
          )}
        </>
      )}

      <div className="mt-6 space-y-4">
        <SellerFollowSection listingId={id} />
        <ReserveListingSection listingId={id} />
        <StartConversationSection listingId={id} />
        <ReportListingSection listingId={id} />
      </div>
    </div>
  );
}

function Gallery({ listing }: { listing: Listing }) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState(0);
  const image = listing.images[selected];

  return (
    <div>
      {/* contain, not cover: cropping the thing being sold is the one thing a
          detail view must not do. The box is reserved either way, so the page
          does not jump when the file lands. */}
      <div className="bg-surface-muted flex aspect-[4/3] w-full items-center justify-center overflow-hidden rounded-lg">
        {image ? (
          // The photo is the product here, so it carries a real description -
          // unlike the thumbnails below, which are a way to change this one.
          <img src={image.url} alt={listing.title} className="h-full w-full object-contain" />
        ) : (
          <svg className="text-muted h-16 w-16 fill-current opacity-50" aria-hidden="true">
            <use href="/icons.svg#brand-mark" />
          </svg>
        )}
      </div>

      {listing.images.length > 1 && (
        <ul className="mt-3 flex gap-2 overflow-x-auto">
          {listing.images.map((thumb, index) => (
            <li key={thumb.id}>
              <button
                type="button"
                onClick={() => setSelected(index)}
                aria-label={t("pages.listingDetail.showPhoto", { n: index + 1 })}
                aria-current={index === selected ? "true" : undefined}
                className={`focus-visible:ring-accent block cursor-pointer overflow-hidden rounded border-2 focus:outline-none focus-visible:ring-2 ${
                  index === selected ? "border-accent" : "border-transparent"
                }`}
              >
                <img src={thumb.url} alt="" loading="lazy" className="h-16 w-16 object-cover" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
