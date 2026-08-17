import { useCallback, useEffect, useRef, useState } from "react";

export type GalleryImageStatus = "pending" | "uploading" | "uploaded" | "error";

export interface GalleryImage {
  /** Stable client-side id, safe to use as a React key. */
  id: string;
  /** The original file, kept around so a later "upload" step can send it. */
  file: File;
  /** Local blob URL for instant preview, revoked once the image is removed/unmounted. */
  previewUrl: string;
  status: GalleryImageStatus;
  error?: string;
  /** Populated once a real upload succeeds against the backend. */
  serverId?: number;
  serverUrl?: string;
}

export const DEFAULT_ACCEPTED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"];
export const DEFAULT_MAX_IMAGE_BYTES = 5 * 1024 * 1024; // 5 MiB — mirrors backend default

interface UseImageGalleryOptions {
  /** Max number of images this gallery can hold at once. */
  maxFiles?: number;
  maxSizeBytes?: number;
  acceptedTypes?: string[];
  /** Called (possibly multiple times) whenever a file is rejected. */
  onError?: (message: string) => void;
}

function makeId(file: File) {
  return `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(36).slice(2)}`;
}

/**
 * Client-side state for a set of images being added/removed/uploaded.
 * Purely local — callers decide if/when to actually POST files anywhere.
 */
export function useImageGallery({
  maxFiles = Infinity,
  maxSizeBytes = DEFAULT_MAX_IMAGE_BYTES,
  acceptedTypes = DEFAULT_ACCEPTED_IMAGE_TYPES,
  onError,
}: UseImageGalleryOptions = {}) {
  const [images, setImages] = useState<GalleryImage[]>([]);
  const objectUrls = useRef<Set<string>>(new Set());

  const addFiles = useCallback(
    (files: File[]) => {
      setImages((prev) => {
        const remaining = maxFiles - prev.length;
        if (remaining <= 0) {
          onError?.(`You can only add up to ${maxFiles} image${maxFiles === 1 ? "" : "s"}.`);
          return prev;
        }

        const next: GalleryImage[] = [];
        for (const file of files) {
          if (next.length >= remaining) {
            onError?.(`You can only add up to ${maxFiles} image${maxFiles === 1 ? "" : "s"}.`);
            break;
          }
          if (!acceptedTypes.includes(file.type)) {
            onError?.(`"${file.name}" isn't a supported image type (use JPEG, PNG or WebP).`);
            continue;
          }
          if (file.size > maxSizeBytes) {
            onError?.(
              `"${file.name}" is larger than ${Math.round(maxSizeBytes / (1024 * 1024))}MB.`,
            );
            continue;
          }

          const previewUrl = URL.createObjectURL(file);
          objectUrls.current.add(previewUrl);
          next.push({ id: makeId(file), file, previewUrl, status: "pending" });
        }
        return next.length ? [...prev, ...next] : prev;
      });
    },
    [maxFiles, maxSizeBytes, acceptedTypes, onError],
  );

  const removeImage = useCallback((id: string) => {
    setImages((prev) => {
      const target = prev.find((img) => img.id === id);
      if (target && objectUrls.current.has(target.previewUrl)) {
        URL.revokeObjectURL(target.previewUrl);
        objectUrls.current.delete(target.previewUrl);
      }
      return prev.filter((img) => img.id !== id);
    });
  }, []);

  const updateImage = useCallback((id: string, patch: Partial<GalleryImage>) => {
    setImages((prev) => prev.map((img) => (img.id === id ? { ...img, ...patch } : img)));
  }, []);

  const clear = useCallback(() => {
    objectUrls.current.forEach((url) => URL.revokeObjectURL(url));
    objectUrls.current.clear();
    setImages([]);
  }, []);

  // revoke any outstanding blob: URLs when the gallery unmounts
  useEffect(() => {
    const urls = objectUrls.current;
    return () => {
      urls.forEach((url) => URL.revokeObjectURL(url));
      urls.clear();
    };
  }, []);

  return { images, addFiles, removeImage, updateImage, clear };
}
