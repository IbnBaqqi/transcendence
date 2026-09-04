import { useModal } from "../../providers/modalContext";
import { Modal } from "./Modal";
import { Chat } from "./Chat";

export function ChatRoot() {
  const { chatOpen, closeChat } = useModal();
  if (!chatOpen) return null;

  return (
    <Modal onClose={closeChat} variant="floating" className="sm:w-96">
      <Chat />
    </Modal>
  );
}
