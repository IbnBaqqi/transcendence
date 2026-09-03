import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { useCategories, flattenCategories, useLocalizedCategoryNames } from "../../api/categories";

type FilterPanelProps = {
  params: URLSearchParams;
  onChange: (key: string, value: string) => void;
};

// Deny-list so a new filter key gets a chip without this list needing an edit.
const NOT_A_CHIP = new Set(["page", "limit", "sort"]);

function useDebouncedText(value: string, onCommit: (value: string) => void, delay = 300) {
  const [draft, setDraft] = useState(value);
  // Adjusting state during render (not an effect) for an external value sync:
  // https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes
  const [synced, setSynced] = useState(value);
  if (value !== synced) {
    setSynced(value);
    setDraft(value);
  }

  const onCommitRef = useRef(onCommit);
  useEffect(() => {
    onCommitRef.current = onCommit;
  });

  useEffect(() => {
    // draft === value here means this run is the sync above, not a keystroke.
    if (draft === value) return;
    const id = setTimeout(() => onCommitRef.current(draft), delay);
    return () => clearTimeout(id);
  }, [draft, value, delay]);

  return [draft, setDraft] as const;
}

function PriceField({
  id,
  label,
  value,
  onCommit,
}: {
  id: string;
  label: string;
  value: string;
  onCommit: (value: string) => void;
}) {
  const [draft, setDraft] = useState(value);
  const [synced, setSynced] = useState(value);
  if (value !== synced) {
    setSynced(value);
    setDraft(value);
  }

  return (
    <div className="flex flex-col gap-1">
      <label className="text-muted text-xs" htmlFor={id}>
        {label}
      </label>
      <input
        id={id}
        type="number"
        min="0"
        step="0.01"
        inputMode="decimal"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => draft !== value && onCommit(draft)}
        className="border-line bg-surface text-foreground w-24 rounded-md border px-3 py-2 text-sm"
      />
    </div>
  );
}

export function FilterPanel({ params, onChange }: FilterPanelProps) {
  const { t } = useTranslation();
  const { data: categories } = useCategories();
  const categoryName = useLocalizedCategoryNames();
  const categoryOptions = flattenCategories(categories ?? []);

  const [keywordDraft, setKeywordDraft] = useDebouncedText(params.get("keyword") ?? "", (v) =>
    onChange("keyword", v),
  );
  const [locationDraft, setLocationDraft] = useDebouncedText(params.get("location") ?? "", (v) =>
    onChange("location", v),
  );

  const chips = [...params.entries()].filter(([key]) => !NOT_A_CHIP.has(key));

  const chipLabel = (key: string, value: string) => {
    if (key === "category") return categoryName(value);
    if (key === "min_price") return t("pages.search.chips.minPrice", { value });
    if (key === "max_price") return t("pages.search.chips.maxPrice", { value });
    return value;
  };

  return (
    <div className="border-line space-y-4 rounded-lg border p-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div className="flex flex-col gap-1">
          <label className="text-muted text-xs" htmlFor="search-keyword">
            {t("pages.search.keywordLabel")}
          </label>
          <input
            id="search-keyword"
            type="text"
            value={keywordDraft}
            onChange={(e) => setKeywordDraft(e.target.value)}
            className="border-line bg-surface text-foreground rounded-md border px-3 py-2 text-sm"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-muted text-xs" htmlFor="search-category">
            {t("pages.search.categoryLabel")}
          </label>
          <select
            id="search-category"
            value={params.get("category") ?? ""}
            onChange={(e) => onChange("category", e.target.value)}
            className="border-line bg-surface text-foreground rounded-md border px-3 py-2 text-sm"
          >
            <option value="">{t("pages.search.allCategories")}</option>
            {categoryOptions.map((option) => (
              <option key={option.slug} value={option.slug}>
                {option.depth === 1 ? `\u00a0\u00a0\u00a0\u00a0${categoryName(option.slug)}` : categoryName(option.slug)}
              </option>
            ))}
          </select>
        </div>

        <div className="flex gap-2">
          <PriceField
            id="search-min-price"
            label={t("pages.search.minPriceLabel")}
            value={params.get("min_price") ?? ""}
            onCommit={(v) => onChange("min_price", v)}
          />
          <PriceField
            id="search-max-price"
            label={t("pages.search.maxPriceLabel")}
            value={params.get("max_price") ?? ""}
            onCommit={(v) => onChange("max_price", v)}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-muted text-xs" htmlFor="search-location">
            {t("pages.search.locationLabel")}
          </label>
          <input
            id="search-location"
            type="text"
            placeholder={t("pages.search.locationPlaceholder")}
            value={locationDraft}
            onChange={(e) => setLocationDraft(e.target.value)}
            className="border-line bg-surface text-foreground rounded-md border px-3 py-2 text-sm"
          />
        </div>
      </div>

      {chips.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {chips.map(([key, value]) => (
            <button
              key={key}
              type="button"
              onClick={() => onChange(key, "")}
              aria-label={t("pages.search.removeFilter", { name: chipLabel(key, value) })}
              className="bg-soft text-soft-contrast hover:bg-soft-hover flex items-center gap-1 rounded-full px-3 py-1 text-xs"
            >
              {chipLabel(key, value)}
              <span aria-hidden="true">×</span>
            </button>
          ))}
          <button
            type="button"
            onClick={() => chips.forEach(([key]) => onChange(key, ""))}
            className="text-accent-active text-xs underline"
          >
            {t("pages.search.clearAll")}
          </button>
        </div>
      )}
    </div>
  );
}
