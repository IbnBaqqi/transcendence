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
      <div className="flex items-center justify-center w-full h-full bg-accent text-accent-contrast font-semibold">
        {initials}
      </div>
      {editable && (
        <div
          className={`absolute bottom-0 left-0 right-0 flex items-center justify-center ${label} bg-black/60 text-white font-medium`}
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
        className={`relative ${circle} rounded-full overflow-hidden ring-2 ring-line select-none
          bg-accent hover:bg-accent-hover active:bg-accent-active
          opacity-90 hover:opacity-100
          transition-colors duration-150
          focus:outline-none focus:ring-2 focus:ring-brand-300 focus:ring-offset-2`}
      >
        {content}
      </button>
    );
  }

  return (
    <div className={`relative ${circle} rounded-full overflow-hidden ring-2 ring-line select-none`}>
      {content}
    </div>
  );
}
