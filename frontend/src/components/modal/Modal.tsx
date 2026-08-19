import { createPortal } from "react-dom";
import { type ReactNode, useEffect } from "react";

type ModalVariant = "dialog" | "floating";

export function Modal({
  onClose,
  children,
  className = "",
  variant = "dialog",
}: {
  onClose: () => void;
  children: ReactNode;
  className?: string;
  variant?: ModalVariant;
}) {
  useEffect(() => {
    // "floating" panels stay open while the page is used: no backdrop to
    // click, no scroll lock, no Escape-to-close. They're dismissed via a
    // close button inside the panel instead.
    if (variant === "floating") return;

    const onEsc = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onEsc);

    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onEsc);
      document.body.style.overflow = prevOverflow;
    };
  }, [onClose, variant]);

  if (variant === "floating") {
    return createPortal(
      <div
        className={`bg-surface fixed right-4 bottom-4 z-50 flex h-[70vh] flex-col overflow-hidden rounded-lg shadow-lg ${className}`}
      >
        {children}
      </div>,
      document.body,
    );
  }

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className={`bg-surface w-full rounded-lg shadow-lg ${className}`}>{children}</div>
    </div>,
    document.body,
  );
}
