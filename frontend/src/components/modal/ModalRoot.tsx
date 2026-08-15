import { useModal } from "../../providers/modalContext";
import { Modal } from "./Modal";
import { LoginModal } from "./LoginModal";
import { ChatModal } from "./ChatModal";
import { ImageUploadModal } from "./ImageUploadModal";

const sizeByModal: Record<string, string> = {
  login: "max-w-sm",
  chat: "max-w-2xl h-[80vh]",
  imageUpload: "max-w-lg",
};

export function ModalRoot() {
  const { activeModal, closeModal } = useModal();
  if (!activeModal) return null;

  return (
    <Modal onClose={closeModal} className={sizeByModal[activeModal]}>
      {activeModal === "login" && <LoginModal />}
      {activeModal === "chat" && <ChatModal />}
      {activeModal === "imageUpload" && <ImageUploadModal />}
    </Modal>
  );
}
