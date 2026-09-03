import type { ListingSearchParams, ListingSort } from "../api/listings";

// Backend caps limit at 50 (maxLimit in backend/internal/service/listing.go).
export const PAGE_SIZE = 20;

// Mirrors the backend's sort allow-list; the literal tuple is what z.enum needs.
export const SORTS = ["newest", "oldest", "price_asc", "price_desc"] as const;

// The backend rejects anything past math.MaxInt32, and a larger number would
// serialise as "1e+30", which it cannot parse either.
const MAX_PAGE = 2_147_483_647;

const CHIP_KEYS = ["keyword", "category", "location", "min_price", "max_price"] as const;

export type FilterKey = (typeof CHIP_KEYS)[number];

// Prices stay strings, as the URL and the inputs hold them.
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

// Normalised so the chip and the API agree: "5.50" and "0x10" would otherwise
// display as typed. Not a number at all is dropped - "NaN" would be a 400.
function parsePrice(raw: string | null): string {
  if (raw === null || raw.trim() === "") return "";
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 ? String(value) : "";
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
    page: Number.isInteger(page) && page > 0 && page <= MAX_PAGE ? page : 1,
  };
}

// The only updater, so "change a filter, go back to page 1" lives in one place.
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

// Empty strings are passed on: toQueryString already drops them.
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
