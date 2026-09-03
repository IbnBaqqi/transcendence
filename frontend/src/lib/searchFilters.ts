import type { ListingSearchParams, ListingSort } from "../api/listings";

// Backend caps limit at 50 (maxLimit in backend/internal/service/listing.go).
export const PAGE_SIZE = 20;

// Mirrors the backend's sort allow-list; the literal tuple is what z.enum needs.
export const SORTS = ["newest", "oldest", "price_asc", "price_desc"] as const;

const CHIP_KEYS = ["keyword", "category", "location", "min_price", "max_price"] as const;

export type FilterKey = (typeof CHIP_KEYS)[number];

// Prices stay strings here, as the URL and the inputs hold them. The number
// conversion happens once, in toSearchQuery.
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

// A hand-edited ?min_price=abc would otherwise reach the API as "NaN" and 400.
function parsePrice(raw: string | null): string {
  if (raw === null || raw.trim() === "") return "";
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 ? raw.trim() : "";
}

export function parseFilters(sp: URLSearchParams): SearchFilters {
  const sort = sp.get("sort") as ListingSort | null;
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

// The only updater: page falls back to 1 unless the patch names one, so
// "change a filter, go back to page 1" lives in a single place.
export function withFilters(prev: SearchFilters, patch: Partial<SearchFilters>): SearchFilters {
  return { ...prev, ...patch, page: patch.page ?? 1 };
}

export function filtersToParams(f: SearchFilters): URLSearchParams {
  const sp = new URLSearchParams();

  for (const key of CHIP_KEYS) {
    if (f[key] !== "") sp.set(key, f[key]);
  }
  if (f.sort !== "newest") sp.set("sort", f.sort);
  if (f.page !== 1) sp.set("page", String(f.page));

  return sp;
}

// Empty strings are passed on rather than stripped: toQueryString in
// api/listings.ts already drops them.
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

export function activeFilters(f: SearchFilters): { key: FilterKey; value: string }[] {
  return CHIP_KEYS.filter((key) => f[key] !== "").map((key) => ({ key, value: f[key] }));
}
