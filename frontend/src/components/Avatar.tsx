type AvatarProps = {
  size?: "sm" | "md" | "lg";
  initials?: string;
  editable?: boolean;
};

const sizeStyles = {
  sm: { circle: "w-8 h-8 text-xs", label: "h-3 text-[8px]" },
  md: { circle: "w-12 h-12 text-base", label: "h-4 text-[9px]" },
  lg: { circle: "w-16 h-16 text-xl", label: "h-5 text-[10px]" },
};

export default function Avatar({
  size = "md",
  initials = "?",
  editable = false,
  onEditClick,
}: AvatarProps) {
  const { circle, label } = sizeStyles[size];

  const content = (
    <>
      <div className="bg-accent text-accent-contrast flex h-full w-full items-center justify-center font-semibold">
        {initials}
      </div>
      {editable && (
        <div
          className={`absolute right-0 bottom-0 left-0 flex items-center justify-center ${label} bg-black/60 font-medium text-white`}
        >
          Edit
        </div>
      )}
    </>
  );

  if (editable) {
    return (
      <button
        type="button"
        onClick={onEditClick}
        aria-label="Edit profile picture"
        className={`relative ${circle} ring-line bg-accent hover:bg-accent-hover active:bg-accent-active focus:ring-brand-300 overflow-hidden rounded-full opacity-90 ring-2 transition-colors duration-150 select-none hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-none`}
      >
        {content}
      </button>
    );
  }

  return (
    <div className={`relative ${circle} ring-line overflow-hidden rounded-full ring-2 select-none`}>
      {content}
    </div>
  );
}
