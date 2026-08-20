import { useState, type ReactNode } from "react";
import { ModalContext, type DialogType, type ImageUploadModalOptions } from "./modalContext";

export function ModalProvider({ children }: { children: ReactNode }) {
  const [activeModal, setActiveModal] = useState<DialogType>(null);
  const [chatOpen, setChatOpen] = useState(false);
  const [imageUploadOptions, setImageUploadOptions] = useState<ImageUploadModalOptions | null>(
    null,
  );

  const openModal = (modal: DialogType, options?: ImageUploadModalOptions) => {
    setActiveModal(modal);
    setImageUploadOptions(modal === "imageUpload" ? (options ?? null) : null);
  };

  const closeModal = () => {
    setActiveModal(null);
    setImageUploadOptions(null);
  };

  return (
    <ModalContext.Provider
      value={{
        activeModal,
        chatOpen,
        imageUploadOptions,
        openModal,
        closeModal,
        openChat: () => setChatOpen(true),
        closeChat: () => setChatOpen(false),
      }}
    >
      {children}
    </ModalContext.Provider>
  );
}
