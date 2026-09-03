import { useCallback, useEffect, useRef, useState } from "react";
import i18next from "../i18n";

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
  /** 0-100 while uploading; absent when the total size is unknown. */
  progress?: number;
  /** The row's UUID once the upload succeeds - what DELETE needs. */
  serverId?: string;
  /** Set alongside `error`: whether offering a retry would be honest. */
  retryable?: boolean;
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
  const imagesRef = useRef<GalleryImage[]>([]);

  useEffect(() => {
    imagesRef.current = images;
  }, [images]);

  const addFiles = useCallback(
    (files: File[]) => {
      const remaining = maxFiles - imagesRef.current.length;
      if (remaining <= 0) {
        onError?.(i18next.t("validation.maxImages", { count: maxFiles, max: maxFiles }));
        return;
      }

      const next: GalleryImage[] = [];
      for (const file of files) {
        if (next.length >= remaining) {
          onError?.(i18next.t("validation.maxImages", { count: maxFiles, max: maxFiles }));
          break;
        }
        if (!acceptedTypes.includes(file.type)) {
          onError?.(i18next.t("validation.unsupportedImageType", { name: file.name }));
          continue;
        }
        if (file.size > maxSizeBytes) {
          onError?.(
            i18next.t("validation.fileTooLarge", {
              name: file.name,
              mb: Math.round(maxSizeBytes / (1024 * 1024)),
            }),
          );
          continue;
        }

        const previewUrl = URL.createObjectURL(file);
        objectUrls.current.add(previewUrl);
        next.push({ id: makeId(file), file, previewUrl, status: "pending" });
      }
      if (next.length) {
        setImages((prev) => [...prev, ...next]);
      }
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
