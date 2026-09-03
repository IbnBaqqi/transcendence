// The browse filters, as a value. Everything here is pure: the URL goes in,
// a plain object comes out, and nothing touches React. That's what lets the
// rules below be tested without rendering anything.
import type { ListingSearchParams, ListingSort } from "../api/listings";

// The backend caps limit at 50 (maxLimit in backend/internal/service/listing.go).
// Exported so the pages and the pagination control can't disagree about it.
export const PAGE_SIZE = 20;

// Mirrors the sort allow-list the backend validates against. A value not in
// here is a 400, so we coerce rather than forward it.
const SORTS: readonly ListingSort[] = ["newest", "oldest", "price_asc", "price_desc"];

// The filters a chip can represent: the ones that are empty by default and get
// dismissed one at a time. sort and page always have a value and both have
// their own visible control, so they're not in here.
const CHIP_KEYS = ["keyword", "category", "location", "min_price", "max_price"] as const;

// `typeof X[number]` turns a const array into the union of its members, so the
// list above is the single place a new filter gets added.
export type FilterKey = (typeof CHIP_KEYS)[number];

// Prices are strings, not numbers: that's what the URL holds and what an input
// holds. Converting happens once, at the API boundary (toSearchQuery), so no
// NaN can ever live in this object.
export interface SearchFilters {
  keyword: string;
  category: string;
  location: string;
  min_price: string;
  max_price: string;
  sort: ListingSort;
  page: number;
}

export const emptyFilters: SearchFilters = {
  keyword: "",
  category: "",
  location: "",
  min_price: "",
  max_price: "",
  sort: "newest",
  page: 1,
};

// A hand-edited ?min_price=abc would otherwise reach the API as the literal
// string "NaN" and come back a 400. Dropping it is the same treatment sort and
// page get below. A too-long keyword is left alone on purpose: that's a real
// value the backend rejects honestly, not a nonsense param we invented.
function parsePrice(raw: string | null): string {
  if (raw === null || raw.trim() === "") return "";
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 ? raw.trim() : "";
}

export function parseFilters(sp: URLSearchParams): SearchFilters {
  const sort = sp.get("sort") as ListingSort | null;
  // Number(null) and Number("") are both 0, so a missing page falls through
  // to 1 without a special case.
  const page = Number(sp.get("page"));

  return {
    keyword: sp.get("keyword") ?? "",
    category: sp.get("category") ?? "",
    location: sp.get("location") ?? "",
    min_price: parsePrice(sp.get("min_price")),
    max_price: parsePrice(sp.get("max_price")),
    sort: sort !== null && SORTS.includes(sort) ? sort : "newest",
    page: Number.isInteger(page) && page > 0 ? page : 1,
  };
}

/**
 * The only way filters get updated. `page` falls back to 1 unless the patch
 * names it, so "change a filter, go back to page 1" is one rule in one place
 * instead of a thing every call site has to remember.
 */
export function withFilters(prev: SearchFilters, patch: Partial<SearchFilters>): SearchFilters {
  return { ...prev, ...patch, page: patch.page ?? 1 };
}

// Defaults are left out: an untouched search is `/search`, not
// `/search?sort=newest&page=1`, and the two still parse to the same filters.
export function filtersToParams(f: SearchFilters): URLSearchParams {
  const sp = new URLSearchParams();

  for (const key of CHIP_KEYS) {
    if (f[key] !== "") sp.set(key, f[key]);
  }
  if (f.sort !== "newest") sp.set("sort", f.sort);
  if (f.page !== 1) sp.set("page", String(f.page));

  return sp;
}

// The API boundary. Empty strings are passed through rather than stripped -
// toQueryString in api/listings.ts already drops "" and undefined alike, so
// re-doing it here would be a second copy of the same rule.
export function toSearchQuery(f: SearchFilters): ListingSearchParams {
  return {
    keyword: f.keyword,
    category: f.category,
    location: f.location,
    min_price: f.min_price === "" ? undefined : Number(f.min_price),
    max_price: f.max_price === "" ? undefined : Number(f.max_price),
    sort: f.sort,
    page: f.page,
    limit: PAGE_SIZE,
  };
}

/**
 * What the dismissible chips render. Derived, never stored: if the chips were
 * their own state, "remove the chip" and "remove the filter" would be two
 * operations that have to agree forever.
 */
export function activeFilters(f: SearchFilters): { key: FilterKey; value: string }[] {
  return CHIP_KEYS.filter((key) => f[key] !== "").map((key) => ({ key, value: f[key] }));
}
