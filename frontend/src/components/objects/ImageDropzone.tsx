import { useRef, useState, type ChangeEvent, type DragEvent, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";
import { Skeleton } from "./Skeleton";
import type { GalleryImage } from "../../hooks/useImageGallery";

type ImageDropzoneProps = {
  images: GalleryImage[];
  onFilesSelected: (files: File[]) => void;
  onRemove: (id: string) => void;
  onRetry?: (id: string) => void;
  /** Allow selecting/dropping more than one file at a time. */
  multiple?: boolean;
  disabled?: boolean;
  accept?: string;
  /** Shown inside the empty-state placeholder, in place of the skeleton. */
  emptyMessage?: string;
  helperText?: string;
  /** Size classes (e.g. "h-56 w-full") for the empty-state placeholder box. */
  emptyBoxClassName?: string;
  /** Size classes (e.g. "h-28 w-28") applied to each thumbnail once images exist. */
  thumbnailClassName?: string;
  shape?: "square" | "circle";
  className?: string;
};

/**
 * Drag-and-drop + click-to-browse image picker. Renders a Skeleton
 * placeholder when there are no images yet, and swaps to a thumbnail
 * grid (each with its own status/remove/retry controls) once files are
 * added. Purely presentational — upload state and any network calls are
 * owned by the caller (see useImageGallery).
 */
export function ImageDropzone({
  images,
  onFilesSelected,
  onRemove,
  onRetry,
  multiple = false,
  disabled = false,
  accept = "image/jpeg,image/png,image/webp",
  emptyMessage,
  helperText,
  emptyBoxClassName = "h-56 w-full",
  thumbnailClassName = "h-28 w-28",
  shape = "square",
  className = "",
}: ImageDropzoneProps) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragActive, setIsDragActive] = useState(false);
  const [isHover, setIsHover] = useState(false);
  const empty = emptyMessage ?? t("dropzone.noPhotos");
  const helper = helperText ?? t("dropzone.dragDrop");

  const openFileBrowser = () => {
    if (disabled) return;
    inputRef.current?.click();
  };

  const handleFiles = (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList).filter((f) => f.type.startsWith("image/"));
    if (files.length === 0) return;
    onFilesSelected(multiple ? files : files.slice(0, 1));
  };

  const handleInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    handleFiles(event.target.files);
    // reset so selecting the same file again still fires onChange
    event.target.value = "";
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(false);
    if (disabled) return;
    handleFiles(event.dataTransfer.files);
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!disabled) setIsDragActive(true);
  };

  const handleDragLeave = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openFileBrowser();
    }
  };

  const roundedClass = shape === "circle" ? "rounded-full" : "rounded-md";
  const canAddMore = multiple || images.length === 0;

  return (
    <div className={className}>
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={multiple}
        className="hidden"
        onChange={handleInputChange}
        disabled={disabled}
      />

      {images.length === 0 ? (
        <div
          role="button"
          tabIndex={disabled ? -1 : 0}
          onClick={openFileBrowser}
          onKeyDown={handleKeyDown}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          aria-label={t("dropzone.ariaUploadPhotos")}
          aria-disabled={disabled}
          className={`relative ${emptyBoxClassName} ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}
        >
          <Skeleton
            className={`h-full w-full ${roundedClass} border-2 border-dashed transition-colors ${
              isDragActive ? "border-accent" : "border-line"
            }`}
            onMouseEnter={() => setIsHover(true)}
            onMouseLeave={() => setIsHover(false)}
          />
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1 px-4 text-center">
            <span className="text-foreground text-sm font-medium">{empty}</span>
            <span
              className={`${isHover ? "text-berry-500" : "text-muted"} text-xs transition-colors duration-150`}
            >
              {helper}
            </span>
          </div>
        </div>
      ) : (
        <div
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          className={`flex flex-wrap gap-3 rounded-md transition-colors ${
            isDragActive ? "outline-accent outline-2 outline-dashed" : ""
          }`}
        >
          {images.map((image) => (
            <div
              key={image.id}
              className={`relative overflow-hidden ${roundedClass} ${thumbnailClassName} bg-surface-soft`}
            >
              <img src={image.previewUrl} alt="" className="h-full w-full object-cover" />

              {image.status === "uploading" && (
                <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-black/40 px-2">
                  {image.progress === undefined ? (
                    <span className="text-xs font-medium text-white">
                      {t("dropzone.uploading")}
                    </span>
                  ) : (
                    <>
                      <div
                        role="progressbar"
                        aria-label={t("dropzone.uploading")}
                        aria-valuenow={image.progress}
                        aria-valuemin={0}
                        aria-valuemax={100}
                        className="h-1 w-full overflow-hidden rounded-full bg-white/30"
                      >
                        <div
                          className="bg-accent h-full transition-[width] duration-150"
                          style={{ width: `${image.progress}%` }}
                        />
                      </div>
                      <span className="text-xs font-medium text-white">{image.progress}%</span>
                    </>
                  )}
                </div>
              )}

              {image.status === "error" && (
                <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-black/60 p-1 text-center">
                  <span className="text-xs font-medium text-white">
                    {image.error ?? t("dropzone.uploadFailed")}
                  </span>
                  {onRetry && image.retryable && (
                    <button
                      type="button"
                      onClick={() => onRetry(image.id)}
                      className="text-xs font-semibold text-white underline"
                    >
                      {t("dropzone.retry")}
                    </button>
                  )}
                </div>
              )}

              {!disabled && (
                <button
                  type="button"
                  onClick={() => onRemove(image.id)}
                  aria-label={t("dropzone.removePhoto")}
                  className="absolute top-1 right-1 flex h-5 w-5 items-center justify-center rounded-full bg-black/60 text-xs leading-none text-white hover:bg-black/80"
                >
                  ×
                </button>
              )}
            </div>
          ))}

          {canAddMore && !disabled && (
            <button
              type="button"
              onClick={openFileBrowser}
              aria-label={t("dropzone.addPhoto")}
              className={`text-muted hover:text-foreground hover:border-accent border-line flex items-center justify-center border-2 border-dashed ${roundedClass} ${thumbnailClassName}`}
            >
              +
            </button>
          )}
        </div>
      )}
    </div>
  );
}
