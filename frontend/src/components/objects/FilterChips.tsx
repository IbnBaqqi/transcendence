import { useTranslation } from "react-i18next";

import { useLocalizedCategoryNames } from "../../api/categories";
import { activeFilters, type FilterKey, type SearchFilters } from "../../lib/searchFilters";

export function FilterChips({
  filters,
  onRemove,
}: {
  filters: SearchFilters;
  onRemove: (key: FilterKey) => void;
}) {
  const { t } = useTranslation();
  const categoryName = useLocalizedCategoryNames();
  const chips = activeFilters(filters);

  if (chips.length === 0) return null;

  const label = (key: FilterKey, value: string) => {
    switch (key) {
      case "category":
        return categoryName(value);
      case "min_price":
        return t("pages.search.chips.minPrice", { value });
      case "max_price":
        return t("pages.search.chips.maxPrice", { value });
      case "location":
        return t("pages.search.chips.location", { value });
      default:
        return value;
    }
  };

  return (
    <ul className="flex flex-wrap gap-2">
      {chips.map(({ key, value }) => (
        <li key={key}>
          <button
            type="button"
            onClick={() => onRemove(key)}
            aria-label={t("pages.search.removeFilter", { filter: label(key, value) })}
            className="border-line bg-soft text-soft-contrast hover:bg-soft-hover inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm"
          >
            {label(key, value)}
            <span aria-hidden="true">×</span>
          </button>
        </li>
      ))}
    </ul>
  );
}
