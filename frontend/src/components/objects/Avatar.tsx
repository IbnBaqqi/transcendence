import { useTranslation } from "react-i18next";

import { useModal } from "../../providers/modalContext";

type AvatarProps = {
  size?: "sm" | "md" | "lg";
  initials?: string;
  /** Hover/active/focus-ring styling + edit-modal button. Implies interactive. */
  editable?: boolean;
  /** Adds hover/active/focus-ring styling without triggering the edit modal. */
  interactive?: boolean;
  /** Local preview/blob or remote URL. Falls back to initials when absent. */
  imageUrl?: string;
  /** Called with the file the user picked in the upload modal, once confirmed. */
  onImageSelected?: (file: File) => void;
  /** Removes the stored picture. Omit when there is none - the modal offers
      the button only when it is given one. */
  onRemove?: () => void | Promise<void>;
};

const sizeStyles = {
  sm: { circle: "w-8 h-8 text-xs", label: "h-3 text-[8px]" },
  md: { circle: "w-12 h-12 text-base", label: "h-4 text-[9px]" },
  lg: { circle: "w-16 h-16 text-xl", label: "h-5 text-[10px]" },
};

const interactiveClasses =
  "ring-line bg-accent hover:bg-accent-hover active:bg-accent-active focus:ring-surface-accent opacity-90 transition-colors duration-150 hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-none";

export default function Avatar({
  size = "md",
  initials = "?",
  editable = false,
  interactive = false,
  imageUrl,
  onImageSelected,
  onRemove,
}: AvatarProps) {
  const { t } = useTranslation();
  const { circle, label } = sizeStyles[size];
  const isInteractive = editable || interactive;

  const content = (
    <>
      {imageUrl ? (
        <img src={imageUrl} alt="" className="h-full w-full object-cover" />
      ) : (
        <div className="bg-accent text-accent-contrast flex h-full w-full items-center justify-center font-semibold">
          {initials}
        </div>
      )}
      {editable && (
        <div
          className={`absolute right-0 bottom-0 left-0 flex items-center justify-center ${label} bg-black/60 font-medium text-white`}
        >
          {t("avatar.edit")}
        </div>
      )}
    </>
  );

  if (editable) {
    return (
      <EditableAvatar
        className={`relative ${circle} overflow-hidden rounded-full ring-2 select-none ${interactiveClasses}`}
        onImageSelected={onImageSelected}
        onRemove={onRemove}
      >
        {content}
      </EditableAvatar>
    );
  }

  return (
    <div
      className={`relative ${circle} overflow-hidden rounded-full ring-2 select-none ${
        isInteractive ? interactiveClasses : "ring-line"
      }`}
    >
      {content}
    </div>
  );
}

// The modal context lives here rather than in Avatar so that a plain avatar -
// a listing card, a follow row - does not drag a ModalProvider into every tree
// that renders one.
function EditableAvatar({
  className,
  onImageSelected,
  onRemove,
  children,
}: {
  className: string;
  onImageSelected?: (file: File) => void;
  onRemove?: () => void | Promise<void>;
  children: React.ReactNode;
}) {
  const { t } = useTranslation();
  const { openModal } = useModal();

  return (
    <button
      type="button"
      onClick={() => openModal("imageUpload", { onComplete: onImageSelected, onRemove })}
      aria-label={t("avatar.editAria")}
      className={className}
    >
      {children}
    </button>
  );
}
