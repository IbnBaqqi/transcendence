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
      <h1 className="text-foreground text-2xl font-bold">{t("pages.search.title")}</h1>

      {/* Remounting on a URL change is what reseeds the panel's draft inputs. */}
      <SearchFilterSection
        key={params.toString()}
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

      {/* Unlike Home, no results here is a normal answer rather than a failure. */}
      {listings?.length === 0 && (
        <p role="status" className="text-muted">
          {t("pages.search.noResults")}
        </p>
      )}

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

      {/* page/total_pages come off the response, not the request: the server
          caps limit, so what it says is what actually happened. */}
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
