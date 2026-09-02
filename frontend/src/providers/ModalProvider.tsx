import { useState, type ReactNode } from "react";
import { ModalContext, type DialogType, type ImageUploadModalOptions } from "./modalContext";

export function ModalProvider({ children }: { children: ReactNode }) {
  const [activeModal, setActiveModal] = useState<DialogType>(null);
  const [chatOpen, setChatOpen] = useState(false);
  const [chatConversationId, setChatConversationId] = useState<string | null>(null);
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
        chatConversationId,
        imageUploadOptions,
        openModal,
        closeModal,
        openChat: (conversationId?: string) => {
          setChatConversationId(conversationId ?? null);
          setChatOpen(true);
        },
        // Cleared on close, or reopening from the header would drop you back
        // into whichever thread you were last in.
        closeChat: () => {
          setChatConversationId(null);
          setChatOpen(false);
        },
      }}
    >
      {children}
    </ModalContext.Provider>
  );
}
