import { useModal } from "../../providers/modalContext";
import { LoginSection } from "../forms/LoginSection";

export function LoginModal() {
  const { closeModal } = useModal();

  return (
    <div className="p-6">
      <h2 className="mb-4 text-lg font-semibold">Log in</h2>
      <LoginSection onClose={closeModal} />
    </div>
  );
}
