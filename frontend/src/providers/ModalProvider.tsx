import { useState, type ReactNode } from "react";
import { ModalContext, type DialogType } from "./modalContext";

export function ModalProvider({ children }: { children: ReactNode }) {
  const [activeModal, setActiveModal] = useState<DialogType>(null);
  const [chatOpen, setChatOpen] = useState(false);

  return (
    <ModalContext.Provider
      value={{
        activeModal,
        chatOpen,
        openModal: setActiveModal,
        closeModal: () => setActiveModal(null),
        openChat: () => setChatOpen(true),
        closeChat: () => setChatOpen(false),
      }}
    >
      {children}
    </ModalContext.Provider>
  );
}
