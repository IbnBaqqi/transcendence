import { useFormContext } from "react-hook-form";
import { useFormConfig } from "./FormContext";

type FormTextAreaProps = {
  label?: string;
  name: string;
  placeholder?: string;
  maxLength?: number;
  isEditing?: boolean;
  validateOnChange?: boolean;
};

export function FormTextArea({
  label,
  name,
  placeholder,
  maxLength = 1024,
  isEditing: isEditingProp,
  validateOnChange,
}: FormTextAreaProps) {
  const {
    register,
    watch,
    trigger,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  const isEditing = isEditingProp ?? ctxEditing ?? false;

  const error = errors[name];
  const value = watch(name) ?? "";

  const { onChange: rhfOnChange, ...registerRest } = register(name);

  return (
    <div className="">
      {isEditing ? (
        <>
          {label && <label htmlFor={name}>{label}</label>}
          <textarea
            className="border-line bg-surface text-foreground m-0 field-sizing-content w-full min-w-64 resize-none overflow-y-auto rounded border p-2 shadow focus:outline-none"
            id={name}
            // field-sizing overrides rows where it is supported, so this is
            // invisible on a current browser. It is the fallback that matters:
            // field-sizing is Baseline only since June 2026, and with rows={1}
            // and resize-none an older browser gives a one-line box the user
            // cannot enlarge.
            rows={3}
            maxLength={maxLength}
            placeholder={placeholder}
            {...registerRest}
            onChange={(e) => {
              rhfOnChange(e);
              if (validateOnChange) trigger(name);
            }}
          />
          {(error || value.length > 0) && (
            <div className="text-muted flex justify-between text-xs">
              <span>
                {error && error.type === "max" && (
                  <span className="text-danger">{error.message as string}</span>
                )}
              </span>
              {value.length > 0 && (
                <span>
                  {value.length} / {maxLength}
                </span>
              )}
            </div>
          )}
        </>
      ) : (
        <>
          {/* See FormField: a label with nothing to point at is the bug. */}
          {label && <span>{label}</span>}
          <span>{value}</span>
        </>
      )}
    </div>
  );
}
