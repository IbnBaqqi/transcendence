import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { SearchFilterSection } from "../components/forms/SearchFilterSection";
import { FilterChips } from "../components/objects/FilterChips";
import { ListingCard } from "../components/objects/ListingCard";
import { Pagination } from "../components/objects/Pagination";
import { Skeleton } from "../components/objects/Skeleton";
import { useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import {
  emptyFilters,
  filtersToParams,
  parseFilters,
  toSearchQuery,
  withFilters,
  type SearchFilters,
} from "../lib/searchFilters";

export default function Search() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  // Parsed every render rather than mirrored into state: the URL is the filters.
  const filters = parseFilters(params);

  const update = (patch: Partial<SearchFilters>) =>
    setParams(filtersToParams(withFilters(filters, patch)));

  const { data, isPending, isError, refetch } = useSearchListings(toSearchQuery(filters));
  const categoryName = useLocalizedCategoryNames();
  const listings = data?.items;

  return (
    <div className="mx-auto max-w-6xl space-y-4 px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.search.title")}</h1>

      {/* Remounting reseeds the draft inputs when the filters change from
          outside. page is out of the key: paging must not discard a draft. */}
      <SearchFilterSection
        key={filtersToParams({ ...filters, page: 1 }).toString()}
        initial={filters}
        onApply={update}
        onClear={() => setParams(filtersToParams(emptyFilters))}
      />

      <FilterChips filters={filters} onRemove={(key) => update({ [key]: "" })} />

      {isPending && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-56 w-full" />
          ))}
        </div>
      )}

      {isError && (
        <Skeleton
          variant="error"
          className="h-56 w-full"
          message={t("pages.search.listingsError")}
          onRetry={() => refetch()}
        />
      )}

      {/* Always in the DOM so the announcement lands. total, not items: an empty
          page past the end is not the same as nothing matching. */}
      <p role="status" aria-live="polite" className="text-muted">
        {data?.total === 0 && t("pages.search.noResults")}
      </p>

      {listings && listings.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {listings.map((listing) => (
            <ListingCard
              key={listing.id}
              listing={listing}
              categoryName={categoryName(listing.category)}
            />
          ))}
        </div>
      )}

      {/* Off the response, not the request: the server caps limit. */}
      {data && data.total > 0 && (
        <Pagination
          page={data.page}
          totalPages={data.total_pages}
          total={data.total}
          onPageChange={(page) => update({ page })}
        />
      )}
    </div>
  );
}
