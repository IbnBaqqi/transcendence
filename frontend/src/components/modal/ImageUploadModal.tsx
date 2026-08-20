import { useModal } from "../../providers/modalContext";
import { ImageUploadSection } from "../forms/ImageUploadSection";

export function ImageUploadModal() {
  const { closeModal, imageUploadOptions } = useModal();

  return (
    <div className="p-6">
      <h2 className="mb-4 text-lg font-semibold">Upload image</h2>
      <ImageUploadSection onComplete={imageUploadOptions?.onComplete} onClose={closeModal} />
    </div>
  );
}
