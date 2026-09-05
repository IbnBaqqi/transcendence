import { useModal } from "../../providers/modalContext";
import { ImageUploadSection } from "../forms/ImageUploadSection";
import { useTranslation } from "react-i18next";

export function ImageUploadModal() {
  const { t } = useTranslation();
  const { closeModal, imageUploadOptions } = useModal();

  return (
    <div className="p-6">
      <h2 className="text-section mb-4 font-semibold">{t("modal.imageUpload.title")}</h2>
      <ImageUploadSection
        onComplete={imageUploadOptions?.onComplete}
        onRemove={imageUploadOptions?.onRemove}
        onClose={closeModal}
      />
    </div>
  );
}
