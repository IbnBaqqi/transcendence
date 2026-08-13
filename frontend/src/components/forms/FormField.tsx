import { useFormContext } from "react-hook-form";

type FormFieldProps = {
  label: string;
  name: string;
  type?: string;
  placeholder?: string;
};

export function FormField({ label, name, type = "text", placeholder }: FormFieldProps) {
  const {
    register,
    formState: { errors },
  } = useFormContext();

  const error = errors[name];

  return (
    <div className="form-field">
      <label htmlFor={name}>{label}</label>
      <input id={name} type={type} placeholder={placeholder} {...register(name)} />
      {error && <span className="form-error">{error.message as string}</span>}
    </div>
  );
}
