import { useFormContext } from "react-hook-form";

import { useCategories, flattenCategories } from "../../api/categories";
import { useFormConfig } from "./FormContext";

type FormSelectProps = {
  label?: string;
  name: string;
  isEditing?: boolean;
  width?: string;
};

export function FormSelect({
  label,
  name,
  isEditing: isEditingProp,
  width: widthProp,
}: FormSelectProps) {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  const { data: categories, isPending, isError } = useCategories();

  const width = widthProp ?? "w-full";
  const isEditing = isEditingProp ?? ctxEditing ?? false;

  const error = errors[name];
  const value = watch(name) ?? "";

  const options = flattenCategories(categories ?? []);
  const selected = options.find((option) => option.slug === value);

  if (!isEditing) {
    return (
      <div className="flex flex-col gap-1.5">
        {label && <label className="text-muted">{label}</label>}
        <span>{selected?.name ?? value}</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-muted" htmlFor={name}>
          {label}
        </label>
      )}

      <select
        className={`focus:shadow-outline ${width} appearance-none rounded border px-3 py-2 leading-tight shadow focus:outline-none`}
        id={name}
        disabled={isPending || isError}
        {...register(name)}
      >
        <option value="">
          {isPending
            ? "Loading categories…"
            : isError
              ? "Categories unavailable"
              : "Choose a category"}
        </option>

        {options.map((option) => (
          <option key={option.slug} value={option.slug}>
            {option.depth === 1 ? `\u00a0\u00a0\u00a0\u00a0${option.name}` : option.name}
          </option>
        ))}
      </select>

      {isError && (
        <span className="text-berry-500 text-xs">
          Could not load categories. Reload the page to try again
        </span>
      )}
      {error && <span className="text-berry-500 text-xs">{error.message as string}</span>}
    </div>
  );
}
