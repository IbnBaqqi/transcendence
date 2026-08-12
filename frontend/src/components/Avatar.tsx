type AvatarProps = {
  size?: "sm" | "md" | "lg";
  initials?: string;
};

const sizeStyles = {
  sm: "w-8 h-8 text-xs",
  md: "w-12 h-12 text-base",
  lg: "w-16 h-16 text-xl",
};

export default function Avatar({
  size = "md",
  initials = "?"
}: AvatarProps) {
  return (
    <div className={`flex items-center justify-center ${sizeStyles[size]} rounded-full bg-surface-accent text-accent-contrast font-semibold ring-2 ring-line select-none`}>
      {initials}
      {/*just placeholder for now, replace with API call for user initials later*/}
    </div>
  );
}
