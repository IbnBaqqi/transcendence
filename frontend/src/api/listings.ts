import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { Listing, ListingImage, Paginated } from "./types";

// The runtime list is the source of truth and the type is derived from it, so
// validating an unknown string and typing a known one can't disagree.
// `as const` freezes it to the literal strings instead of widening to string[].
export const LISTING_SORTS = ["newest", "oldest", "price_asc", "price_desc"] as const;

// Mirrors the backend's sort allow-list, so a typo is a compile error rather
// than a 400 at runtime.
export type ListingSort = (typeof LISTING_SORTS)[number];

// One page of results. Home and User currently hardcode this; they adopt the
// constant when they get pagination controls.
export const PAGE_SIZE = 20;

export interface ListingSearchParams {
  keyword?: string;
  category?: string;
  tag?: string;
  seller_id?: string;
  min_price?: number;
  max_price?: number;
  location?: string;
  sort?: ListingSort;
  page?: number;
  limit?: number;
}

// Sorted, so the same filters always produce the same cache key regardless of
// the order the caller wrote them in.
export function toQueryString(params: ListingSearchParams): string {
  const search = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") search.set(key, String(value));
  }
  search.sort();

  return search.toString();
}

// The other direction: a URL's query string back into typed params.
//
// This is the trust boundary. Anything in `?...` was typed, pasted or edited by
// a person, so a value the backend would reject is dropped here rather than
// forwarded for a 400 - a bad ?sort= should show unsorted results, not an error.
export function toSearchParams(search: URLSearchParams): ListingSearchParams {
  // Number("") and Number(null) are both 0, so the empty cases go first.
  const positive = (key: string): number | undefined => {
    const raw = search.get(key);
    if (raw === null || raw.trim() === "") return undefined;
    const value = Number(raw);
    return Number.isFinite(value) && value >= 0 ? value : undefined;
  };

  const text = (key: string) => search.get(key) || undefined;

  const sort = search.get("sort");
  const page = positive("page");

  return {
    keyword: text("keyword"),
    category: text("category"),
    tag: text("tag"),
    location: text("location"),
    min_price: positive("min_price"),
    max_price: positive("max_price"),
    // .includes on a readonly tuple won't accept a plain string, so the cast
    // asks the question; the result of the check is what makes it true.
    sort: LISTING_SORTS.includes(sort as ListingSort) ? (sort as ListingSort) : undefined,
    page: page && page >= 1 ? Math.floor(page) : 1,
    limit: PAGE_SIZE,
  };
}

export function useListing(id: string) {
  return useQuery({
    queryKey: keys.listings.detail(id),
    queryFn: async () => (await api.get<Listing>(apiPath`/listings/${id}`)).data,
    enabled: id !== "", // skip while a route param is still being parsed
  });
}

export function useSearchListings(
  params: ListingSearchParams,
  options: { enabled?: boolean } = {},
) {
  const query = toQueryString(params);

  return useQuery({
    queryKey: keys.listings.search(query),
    queryFn: async () => (await api.get<Paginated<Listing>>(`/listings/search?${query}`)).data,
    // Without a seller_id this searches everything, so callers waiting on one
    // pass false rather than fetching the whole catalogue.
    enabled: options.enabled ?? true,
  });
}

// Listings already carry their images, so this is only for the cases that
// need them on their own - the upload UI in #90.
export interface CreateListingInput {
  title: string;
  description: string;
  category: string;
  price: number;
  quantity: number;
  unit: string;
}

export function useCreateListing() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateListingInput) =>
      (await api.post<Listing>("/listings", input)).data,

    onSuccess: (listing) => {
      queryClient.setQueryData(keys.listings.detail(listing.id), listing);
      return queryClient.invalidateQueries({ queryKey: keys.listings.all });
    },
  });
}

export function useListingImages(id: string) {
  return useQuery({
    queryKey: keys.listings.images(id),
    queryFn: async () =>
      (await api.get<ListingImage[]>(apiPath`/listings/${id}/images`)).data ?? [],
    enabled: id !== "",
  });
}

async function uploadListingImage(listingId: string, file: File): Promise<ListingImage> {
  const body = new FormData();
  body.append("image", file);
  const res = await api.post<ListingImage>(apiPath`/listings/${listingId}/images`, body);
  return res.data;
}

// The id comes per call, not per hook: a listing being created has none until
// POST /listings answers, and that is the only caller today.
export function useUploadListingImage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ listingId, file }: { listingId: string; file: File }) =>
      uploadListingImage(listingId, file),
    onSuccess: (_image, { listingId }) => {
      void queryClient.invalidateQueries({ queryKey: keys.listings.images(listingId) });
      void queryClient.invalidateQueries({ queryKey: keys.listings.detail(listingId) });
    },
  });
}

async function deleteListingImage(listingId: string, imageId: string): Promise<void> {
  await api.delete(apiPath`/listings/${listingId}/images/${imageId}`);
}

export function useDeleteListingImage(listingId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (imageId: string) => deleteListingImage(listingId as string, imageId),
    onSuccess: () => {
      const id = listingId as string;
      queryClient.invalidateQueries({ queryKey: keys.listings.images(id) });
      queryClient.invalidateQueries({ queryKey: keys.listings.detail(id) });
    },
  });
}
