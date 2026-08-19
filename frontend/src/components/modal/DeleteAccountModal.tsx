import { useModal } from "../../providers/modalContext";
import { DeleteAccountSection } from "../forms/DeleteAccountSection";

export function DeleteAccountModal() {
  const { closeModal } = useModal();

  return (
    <div className="p-6">
      <h2 className="mb-4 text-lg font-semibold">Delete Account</h2>
      <DeleteAccountSection onClose={closeModal} />
    </div>
  );
}
