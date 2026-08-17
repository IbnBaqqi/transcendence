import { useFormContext } from "react-hook-form";
import { useFormConfig } from "./FormContext";

type FormTextAreaProps = {
  label?: string;
  name: string;
  placeholder?: string;
  maxLength?: number;
  isEditing?: boolean;
};

export function FormTextArea({
  label,
  name,
  placeholder,
  maxLength = 1024,
  isEditing: isEditingProp,
}: FormTextAreaProps) {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  const isEditing = isEditingProp ?? ctxEditing ?? false;

  const error = errors[name];
  const value = watch(name) ?? "";

  return (
    <div className="">
      {label && <label htmlFor={name}>{label}</label>}
      {isEditing ? (
        <>
          <textarea
            className="field-sizing-content m-0 w-full min-w-64 resize-none overflow-y-auto rounded border p-2 shadow focus:outline-none"
            id={name}
            rows={1}
            maxLength={maxLength}
            placeholder={placeholder}
            {...register(name)}
          />
          {(error || value.length > 0) && (
            <div className="text-muted flex justify-between text-xs">
              <span>
                {error && <span className="text-berry-500">{error.message as string}</span>}
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
        <span>{value}</span>
      )}
    </div>
  );
}
