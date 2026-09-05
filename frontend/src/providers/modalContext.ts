import { createContext, useContext } from "react";

export type DialogType = "login" | "register" | "imageUpload" | "deleteAccount" | null;

// Extra data a caller can hand to a specific modal when opening it. Only
// "imageUpload" needs one today - it hands back the file the user picked
// once they confirm, so e.g. Avatar/Profile can show the new preview.
export interface ImageUploadModalOptions {
  onComplete?: (file: File) => void;
  // Present only when there is a stored picture to remove: the modal shows the
  // button if it is handed one, and asks nothing about the profile itself.
  onRemove?: () => void | Promise<void>;
}

export interface ModalContextValue {
  activeModal: DialogType;
  chatOpen: boolean;
  // The thread to open on, when something opened the chat with one in mind
  // (starting a conversation from a listing). Null means "show the list".
  chatConversationId: string | null;
  imageUploadOptions: ImageUploadModalOptions | null;
  openModal: (modal: DialogType, options?: ImageUploadModalOptions) => void;
  closeModal: () => void;
  openChat: (conversationId?: string) => void;
  closeChat: () => void;
}

export const ModalContext = createContext<ModalContextValue | null>(null);

export function useModal() {
  const ctx = useContext(ModalContext);
  if (!ctx) throw new Error("useModal must be used within ModalProvider");
  return ctx;
}
