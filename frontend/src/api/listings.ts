import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "./client";
import { keys } from "./queryKeys";
import type { Listing, ListingImage, Paginated } from "./types";

// Mirrors the backend's sort allow-list, so a typo is a compile error rather
// than a 400 at runtime.
export type ListingSort = "newest" | "oldest" | "price_asc" | "price_desc";

export interface ListingSearchParams {
  keyword?: string;
  category?: string;
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

export function useListings() {
  return useQuery({
    queryKey: keys.listings.list(),
    queryFn: async () => {
      const res = await api.get<Listing[]>("/listings");
      return res.data ?? []; // insurance: an empty result must not be null
    },
  });
}

export function useListing(id: number) {
  return useQuery({
    queryKey: keys.listings.detail(id),
    queryFn: async () => (await api.get<Listing>(`/listings/${id}`)).data,
    enabled: Number.isInteger(id) && id > 0, // skip while a route param is still being parsed
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
export function useListingImages(id: number) {
  return useQuery({
    queryKey: keys.listings.images(id),
    queryFn: async () => (await api.get<ListingImage[]>(`/listings/${id}/images`)).data ?? [],
    enabled: Number.isInteger(id) && id > 0,
  });
}

async function uploadListingImage(listingId: number, file: File): Promise<ListingImage> {
  const body = new FormData();
  body.append("image", file);
  const res = await api.post<ListingImage>(`/listings/${listingId}/images`, body);
  return res.data;
}

export function useUploadListingImage(listingId: number | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => uploadListingImage(listingId as number, file),
    onSuccess: () => {
      const id = listingId as number;
      queryClient.invalidateQueries({ queryKey: keys.listings.images(id) });
      queryClient.invalidateQueries({ queryKey: keys.listings.detail(id) });
    },
  });
}

async function deleteListingImage(listingId: number, imageId: number): Promise<void> {
  await api.delete(`/listings/${listingId}/images/${imageId}`);
}

export function useDeleteListingImage(listingId: number | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (imageId: number) => deleteListingImage(listingId as number, imageId),
    onSuccess: () => {
      const id = listingId as number;
      queryClient.invalidateQueries({ queryKey: keys.listings.images(id) });
      queryClient.invalidateQueries({ queryKey: keys.listings.detail(id) });
    },
  });
}
