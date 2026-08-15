import { useFormContext } from "react-hook-form";

type FormFieldProps = {
  label?: string;
  name: string;
  type?: string;
  placeholder?: string;
  isEditing: boolean;
};

export function FormField({ label, name, type = "text", placeholder, isEditing }: FormFieldProps) {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext();

  const error = errors[name];
  const value = watch(name) ?? "";

  return (
    <div className="">
      {label && <label htmlFor={name}>{label}</label>}
      {isEditing ? (
        <>
          <input
            className="focus:shadow-outline w-full appearance-none rounded border px-3 py-2 leading-tight shadow focus:outline-none"
            id={name}
            type={type}
            placeholder={placeholder}
            {...register(name)}
          />
          {error && <span className="text-berry-500">{error.message as string}</span>}
        </>
      ) : (
        <span>{value}</span>
      )}
    </div>
  );
}
