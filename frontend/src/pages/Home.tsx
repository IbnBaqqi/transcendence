import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { PAGE_SIZE, useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { ListingCard } from "../components/objects/ListingCard";
import { Pagination } from "../components/objects/Pagination";
import { Skeleton } from "../components/objects/Skeleton";

export default function Home() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Number(searchParams.get("page")) || 1;

  const { data, isPending, isError, refetch } = useSearchListings({ page, limit: PAGE_SIZE });
  const listings = data?.items;
  const categoryName = useLocalizedCategoryNames();

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{t("pages.home.latestFinds")}</h1>
      {/*
        Live region: always in the DOM so assistive tech has something to
        watch. When the text inside changes, screen readers announce it.
        Rendering this <p> conditionally would defeat that - the region and its
        content would appear together, which often goes unannounced.
        aria-live="polite" = wait for a pause, don't interrupt the user.
        When every condition is false it renders empty, taking no space.
      */}
      {/* Plain text Loading / Error / Message view */}
      {/* <p role="status" className="text-muted mt-4"> */}
      {/*   {isPending && "Loading..."} */}
      {/*   {isError && "Couldn't load listings. Try again."} */}
      {/*   {listings?.length === 0 && "No listings yet!"} */}
      {/* </p> */}
      {isPending && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-56 w-full" />
          ))}
        </div>
      )}
      {isError && (
        <Skeleton
          variant="error"
          className="mt-6 h-56 w-full"
          message={t("pages.home.listingsError")}
          onRetry={() => refetch()}
        />
      )}
      {listings?.length === 0 && (
        <Skeleton
          variant="error"
          className="mt-6 h-56 w-full"
          message={t("pages.home.noListings")}
          onRetry={() => refetch()}
        />
      )}
      {listings && listings.length > 0 && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {listings.map((listing) => (
            <ListingCard
              key={listing.id}
              listing={listing}
              categoryName={categoryName(listing.category)}
            />
          ))}
        </div>
      )}
      {data && (
        <Pagination
          page={data.page}
          totalPages={data.total_pages}
          onChange={(next) => setSearchParams({ page: String(next) }, { replace: true })}
        />
      )}
    </div>
  );
}
