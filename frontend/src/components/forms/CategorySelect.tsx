import { useFormContext } from "react-hook-form";

import { useCategories, flattenCategories } from "../../api/categories";
import { useFormConfig } from "./FormContext";

type CategorySelectProps = {
  label?: string;
  ariaLabel?: string;
  name: string;
  isEditing?: boolean;
  width?: string;
};

export function CategorySelect({
  label,
  ariaLabel,
  name,
  isEditing: isEditingProp,
  width: widthProp,
}: CategorySelectProps) {
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  const { data: categories, isPending, isError, refetch } = useCategories();

  const width = widthProp ?? "w-full";
  const isEditing = isEditingProp ?? ctxEditing ?? false;

  const error = errors[name];
  const value = watch(name) ?? "";

  const options = flattenCategories(categories ?? []);
  const selected = options.find((option) => option.slug === value);
  const empty = !isPending && !isError && options.length === 0;
  const unavailable = isPending || isError || empty;

  const statusId = `${name}-status`;
  const errorId = `${name}-error`;
  const describedBy =
    [!isPending && unavailable ? statusId : null, error ? errorId : null]
      .filter(Boolean)
      .join(" ") || undefined;

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
        className={`focus:shadow-outline ${width} rounded border px-3 py-2 leading-tight shadow focus:outline-none`}
        id={name}
        aria-label={label ? undefined : ariaLabel}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        disabled={unavailable}
        {...register(name)}
      >
        <option value="">
          {isPending
            ? "Loading categories…"
            : unavailable
              ? "Categories unavailable"
              : "Choose a category"}
        </option>

        {options.map((option) => (
          <option key={option.slug} value={option.slug}>
            {option.depth === 1 ? `\u00a0\u00a0\u00a0\u00a0${option.name}` : option.name}
          </option>
        ))}
      </select>

      {empty && (
        <span id={statusId} role="alert" className="text-berry-500 text-xs">
          No categories are available
        </span>
      )}
      {isError && (
        <span id={statusId} role="alert" className="text-berry-500 text-xs">
          Could not load categories.{" "}
          <button type="button" className="underline" onClick={() => void refetch()}>
            Try again
          </button>
        </span>
      )}
      {error && (
        <span id={errorId} role="alert" className="text-berry-500 text-xs">
          {error.message as string}
        </span>
      )}
    </div>
  );
}
