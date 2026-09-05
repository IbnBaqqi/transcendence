import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ImageDropzone } from "../objects/ImageDropzone";
import { useImageGallery } from "../../hooks/useImageGallery";
import Button from "../objects/Button.tsx";

type ImageUploadSectionProps = {
  onComplete?: (file: File) => void;
  onRemove?: () => void | Promise<void>;
  onClose: () => void;
};

// Single-image picker used inside the "imageUpload" modal (currently just the
// avatar flow, see Avatar.tsx/Profile.tsx). It sends nothing itself: the file
// goes back to the caller via onComplete, which decides where it is uploaded.
export function ImageUploadSection({ onComplete, onRemove, onClose }: ImageUploadSectionProps) {
  const { t } = useTranslation();
  const [error, setError] = useState<string | null>(null);
  const { images, addFiles, removeImage } = useImageGallery({
    maxFiles: 1,
    onError: setError,
  });

  const handleFilesSelected = (files: File[]) => {
    setError(null);
    addFiles(files);
  };

  const handleRemove = async () => {
    await onRemove?.();
    onClose();
  };

  const handleSave = () => {
    const image = images[0];
    if (!image) return;
    onComplete?.(image.file);
    onClose();
  };

  return (
    <div className="space-y-4">
      <ImageDropzone
        images={images}
        onFilesSelected={handleFilesSelected}
        onRemove={removeImage}
        emptyMessage={t("dropzone.noPhoto")}
        emptyBoxClassName="h-48 w-48 mx-auto"
        thumbnailClassName="h-48 w-48"
        shape="circle"
      />
      {error && <p className="text-danger text-center text-sm">{error}</p>}
      <div className="flex flex-row gap-2">
        <Button variant="primary" type="button" onClick={handleSave} disabled={images.length === 0}>
          {t("common.save")}
        </Button>
        <Button variant="secondary" type="button" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        {/* Right and tertiary: it acts on the stored picture, not on the file
            being picked, so it should not read as a third step in that flow. */}
        {onRemove && (
          <div className="ml-auto">
            <Button variant="tertiary" type="button" onClick={() => void handleRemove()}>
              {t("avatar.remove")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
