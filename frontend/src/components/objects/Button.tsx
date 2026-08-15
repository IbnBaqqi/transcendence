type ButtonProps = {
  children: React.ReactNode;
  onClick?: () => void; // Insert event handler here
  disabled?: boolean; // Start true or false
  variant?: "primary" | "secondary"; // Variants for main buttons and secondaries
  type?: "button" | "submit" | "reset"; // Native button type, defaults to "button"
};

export default function Button({
  children,
  onClick,
  disabled = false,
  variant = "primary",
  type = "button",
}: ButtonProps) {
  const baseStyles =
    "px-4 py-2 rounded-full font-medium transition-colors duration-150 " +
    "disabled:opacity-50 disabled:cursor-not-allowed " +
    "focus:outline-none focus:ring-2 focus:ring-offset-2";

  const variantStyles = {
    primary:
      "bg-accent text-accent-contrast hover:bg-accent-hover active:bg-accent-active focus:ring-accent",
    secondary:
      "bg-soft text-soft-contrast border border-line hover:bg-soft-hover active:bg-soft-active focus:ring-accent",
  };

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className={`${baseStyles} ${variantStyles[variant]}`}
    >
      {children}
    </button>
  );
}
