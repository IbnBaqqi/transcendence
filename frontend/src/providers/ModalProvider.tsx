import { useState, type ReactNode } from "react";
import { ModalContext, type ModalType } from "./modalContext";

export function ModalProvider({ children }: { children: ReactNode }) {
  const [activeModal, setActiveModal] = useState<ModalType>(null);

  return (
    <ModalContext.Provider
      value={{
        activeModal,
        openModal: setActiveModal,
        closeModal: () => setActiveModal(null),
      }}
    >
      {children}
    </ModalContext.Provider>
  );
}
