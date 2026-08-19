import { createContext, useContext } from "react";

export type DialogType = "login" | "imageUpload" | "deleteAccount" | null;

export interface ModalContextValue {
  activeModal: DialogType;
  chatOpen: boolean;
  openModal: (modal: DialogType) => void;
  closeModal: () => void;
  openChat: () => void;
  closeChat: () => void;
}

export const ModalContext = createContext<ModalContextValue | null>(null);

export function useModal() {
  const ctx = useContext(ModalContext);
  if (!ctx) throw new Error("useModal must be used within ModalProvider");
  return ctx;
}
