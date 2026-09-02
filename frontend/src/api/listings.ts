import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { Listing, ListingImage, Paginated } from "./types";

// Mirrors the backend's sort allow-list, so a typo is a compile error rather
// than a 400 at runtime.
export type ListingSort = "newest" | "oldest" | "price_asc" | "price_desc";

export interface ListingSearchParams {
  keyword?: string;
  category?: string;
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

export function useListing(id: string) {
  return useQuery({
    queryKey: keys.listings.detail(id),
    queryFn: async () => (await api.get<Listing>(apiPath`/listings/${id}`)).data,
    enabled: id !== "", // skip while a route param is still being parsed
  });
}

export function useSearchListings(params: ListingSearchParams) {
  const query = toQueryString(params);

  return useQuery({
    queryKey: keys.listings.search(query),
    queryFn: async () => (await api.get<Paginated<Listing>>(`/listings/search?${query}`)).data,
  });
}

// Listings already carry their images, so this is only for the cases that
// need them on their own - the upload UI in #90.
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

export function useUploadListingImage(listingId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => uploadListingImage(listingId as string, file),
    onSuccess: () => {
      const id = listingId as string;
      queryClient.invalidateQueries({ queryKey: keys.listings.images(id) });
      queryClient.invalidateQueries({ queryKey: keys.listings.detail(id) });
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
