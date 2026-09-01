import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

import { useCategories, flattenCategories, useLocalizedCategoryNames } from "../../api/categories";
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
  const { t } = useTranslation();
  const {
    register,
    watch,
    formState: { errors },
  } = useFormContext();
  const { isEditing: ctxEditing } = useFormConfig();

  const { data: categories, isPending, isError, refetch } = useCategories();
  const categoryName = useLocalizedCategoryNames();

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
        <span>{selected ? categoryName(selected.slug) : value}</span>
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
            ? t("forms.category.loading")
            : unavailable
              ? t("forms.category.unavailable")
              : t("forms.category.choose")}
        </option>

        {options.map((option) => (
          <option key={option.slug} value={option.slug}>
            {option.depth === 1
              ? `\u00a0\u00a0\u00a0\u00a0${categoryName(option.slug)}`
              : categoryName(option.slug)}
          </option>
        ))}
      </select>

      {empty && (
        <span id={statusId} role="alert" className="text-berry-500 text-xs">
          {t("forms.category.noCategories")}
        </span>
      )}
      {isError && (
        <span id={statusId} role="alert" className="text-berry-500 text-xs">
          {t("forms.category.loadError")}{" "}
          <button type="button" className="underline" onClick={() => void refetch()}>
            {t("common.tryAgain")}
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
