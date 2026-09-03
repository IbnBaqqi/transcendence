import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { toSearchParams, useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { FilterPanel } from "../components/objects/FilterPanel";
import { ListingCard } from "../components/objects/ListingCard";
import { Pagination } from "../components/objects/Pagination";
import { Skeleton } from "../components/objects/Skeleton";

export default function Search() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();

  const handleFilterChange = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value);
    else next.delete(key);
    next.delete("page");
    setSearchParams(next);
  };

  const setPage = (next: number) => {
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set("page", String(next));
    setSearchParams(nextParams);
  };

  const params = toSearchParams(searchParams);
  const { data, isPending, isError, refetch } = useSearchListings(params);
  const listings = data?.items;
  const categoryName = useLocalizedCategoryNames();

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{t("pages.search.title")}</h1>

      <div className="mt-4">
        <FilterPanel params={searchParams} onChange={handleFilterChange} />
      </div>

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
          message={t("pages.search.listingsError")}
          onRetry={() => refetch()}
        />
      )}
      {listings?.length === 0 && (
        <Skeleton
          variant="error"
          className="mt-6 h-56 w-full"
          message={t("pages.search.noResults")}
          onRetry={() => refetch()}
        />
      )}
      {data && listings && listings.length > 0 && (
        <>
          <p className="text-muted mt-4 text-sm">
            {t("pages.search.resultCount", { count: data.total })}
          </p>
          <div className="mt-2 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {listings.map((listing) => (
              <ListingCard
                key={listing.id}
                listing={listing}
                categoryName={categoryName(listing.category)}
              />
            ))}
          </div>
          <Pagination page={data.page} totalPages={data.total_pages} onChange={setPage} />
        </>
      )}
    </div>
  );
}
