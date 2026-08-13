type ToggleProps = {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  size?: "sm" | "md";
  label?: string;
};

const sizeStyles = {
  sm: {
    track: "w-9 h-5",
    thumb: "w-4 h-4",
    translate: "translate-x-4",
  },
  md: {
    track: "w-11 h-6",
    thumb: "w-5 h-5",
    translate: "translate-x-5",
  },
};

export default function Toggle({
  checked,
  onChange,
  disabled = false,
  size = "md",
  label,
}: ToggleProps) {
  const { track, thumb, translate } = sizeStyles[size];

  return (
    <label className="inline-flex cursor-pointer items-center gap-2 select-none">
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex ${track} items-center rounded-full transition-colors duration-200 ${checked ? "bg-accent" : "bg-slate-300"} focus:ring-line focus:ring-2 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50`}
      >
        <span
          className={`inline-block ${thumb} transform rounded-full bg-white shadow transition-transform duration-200 ${checked ? translate : "translate-x-0.5"}`}
        />
      </button>
      {label && <span className="text-muted text-base">{label}</span>}
    </label>
  );
}
