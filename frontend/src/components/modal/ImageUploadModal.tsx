import { useModal } from "../../providers/modalContext";
import Button from "../objects/Button.tsx";

export function ImageUploadModal() {
  const { closeModal } = useModal();

  return (
    <div className="p-6">
      <h2 className="mb-4 text-lg font-semibold">Upload image</h2>
      {/* upload UI */}
      <Button variant="secondary" type="button" onClick={closeModal}>
        Cancel
      </Button>
    </div>
  );
}
