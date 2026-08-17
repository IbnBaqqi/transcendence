import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type { ListingImage } from "./types";

// GET /api/v1/listings/{id}/images -> images for a listing, ordered by position
async function fetchListingImages(listingId: number): Promise<ListingImage[]> {
  const res = await api.get<ListingImage[]>(`/listings/${listingId}/images`);
  return res.data ?? [];
}

export function useListingImages(listingId: number | undefined) {
  return useQuery({
    queryKey: ["listingImages", listingId],
    queryFn: () => fetchListingImages(listingId as number),
    enabled: listingId != null,
  });
}

// POST /api/v1/listings/{id}/images -> multipart form, single "image" field.
// One request per file - there's no batch upload endpoint.
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
      queryClient.invalidateQueries({ queryKey: ["listingImages", listingId] });
    },
  });
}

// DELETE /api/v1/listings/{id}/images/{imageID}
async function deleteListingImage(listingId: number, imageId: number): Promise<void> {
  await api.delete(`/listings/${listingId}/images/${imageId}`);
}

export function useDeleteListingImage(listingId: number | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (imageId: number) => deleteListingImage(listingId as number, imageId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["listingImages", listingId] });
    },
  });
}
