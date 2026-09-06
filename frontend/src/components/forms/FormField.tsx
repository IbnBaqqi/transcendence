import { useFormContext } from "react-hook-form";
import { useFormConfig } from "./FormContext";

type FormFieldProps = {
  label?: string;
  name: string;
  type?: string;
  placeholder?: string;
  isEditing?: boolean;
  width?: `max-w-${string}`;
  validateOnChange?: boolean;
  maxLength?: number;
};

export function FormField({
  label,
  name,
  type = "text",
  placeholder,
  isEditing: isEditingProp,
  width: widthProp,
  validateOnChange,
  // A sanity bound, not the validation. It must stay strictly ABOVE every
  // schema cap (the highest is search text at 200): set it equal to one and
  // the field can no longer hold a value that breaks the rule, so zod's error
  // for it becomes unreachable. What this stops is the field growing without
  // bound before anything is submitted.
  maxLength = 512,
}: FormFieldProps) {
  const {
    register,
    watch,
    trigger,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  // The cap applies on top of w-full: a control with only a max-width falls
  // back to its intrinsic width and never reaches the cap.
  const width = widthProp ? `w-full ${widthProp}` : "w-full";
  const isEditing = isEditingProp ?? ctxEditing ?? false;

  const error = errors[name];
  const value = watch(name) ?? "";

  const { onChange: rhfOnChange, ...registerRest } = register(
    name,
    type === "number" ? { valueAsNumber: true } : undefined,
  );

  return (
    <div className="flex flex-col gap-1.5">
      {isEditing ? (
        <>
          {label && (
            <label className="text-muted" htmlFor={name}>
              {label}
            </label>
          )}
          <input
            className={`focus:shadow-outline ${width} appearance-none rounded border px-3 py-2 leading-tight shadow focus:outline-none`}
            id={name}
            type={type}
            maxLength={maxLength}
            placeholder={placeholder}
            {...registerRest}
            onChange={(e) => {
              // The attribute above is ignored on type="number", so cut the
              // raw value before react-hook-form ever reads it.
              if (e.target.value.length > maxLength) {
                e.target.value = e.target.value.slice(0, maxLength);
              }
              rhfOnChange(e);
              if (validateOnChange) trigger(name);
            }}
            onKeyDown={
              type === "number"
                ? (e) => {
                    if (e.key === "e" || e.key === "E") e.preventDefault();
                  }
                : undefined
            }
          />
          {error && <span className="text-danger text-xs">{error.message as string}</span>}
        </>
      ) : (
        <>
          {/* A span, not a label: no input is rendered here, so `for` would
              point at nothing - and a label with neither `for` nor a nested
              control is the same DevTools issue by another name. */}
          {label && <span className="text-muted">{label}</span>}
          <span>{value}</span>
        </>
      )}
    </div>
  );
}
