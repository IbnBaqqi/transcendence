import { useForm, type DefaultValues } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";

import { Form } from "./Form";
import { FormField } from "./FormField";
import { CategorySelect } from "./CategorySelect";
import Button from "../objects/Button";
import { searchFiltersSchema, type SearchFiltersFormValues } from "../../schemas/searchFilters";
import { SORTS, emptyFilters, type SearchFilters } from "../../lib/searchFilters";

// The form works in numbers, the URL in strings. An unset price is undefined
// rather than NaN: react-hook-form writes a default straight into the DOM, and
// "NaN" is not a value type="number" can parse. NaN still means "unset" on the
// way back out - an emptied number input submits as NaN through valueAsNumber.
function toFormValues(f: SearchFilters): DefaultValues<SearchFiltersFormValues> {
  return {
    keyword: f.keyword,
    category: f.category,
    location: f.location,
    min_price: f.min_price === "" ? undefined : Number(f.min_price),
    max_price: f.max_price === "" ? undefined : Number(f.max_price),
    sort: f.sort,
  };
}

function toFilterPatch(v: SearchFiltersFormValues): Partial<SearchFilters> {
  return {
    keyword: v.keyword,
    category: v.category,
    location: v.location,
    min_price: Number.isNaN(v.min_price) ? "" : String(v.min_price),
    max_price: Number.isNaN(v.max_price) ? "" : String(v.max_price),
    sort: v.sort,
  };
}

type SearchFilterSectionProps = {
  initial: SearchFilters;
  onApply: (patch: Partial<SearchFilters>) => void;
  onClear: () => void;
};

export function SearchFilterSection({ initial, onApply, onClear }: SearchFilterSectionProps) {
  const { t } = useTranslation();

  const form = useForm<SearchFiltersFormValues>({
    resolver: zodResolver(searchFiltersSchema),
    defaultValues: toFormValues(initial),
  });

  // Remounting is what reseeds this panel, and clearing an empty URL is no change.
  const handleClear = () => {
    form.reset(toFormValues(emptyFilters));
    onClear();
  };

  return (
    <Form
      form={form}
      onSubmit={(values) => onApply(toFilterPatch(values))}
      isEditing
      className="border-line bg-surface space-y-4 rounded-lg border p-4"
    >
      <h2 className="text-foreground text-section font-bold">{t("pages.search.filters")}</h2>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <FormField name="keyword" label={t("pages.search.keyword")} />
        <CategorySelect name="category" label={t("pages.search.category")} />
        <FormField name="location" label={t("pages.search.location")} />
        <FormField name="min_price" type="number" label={t("pages.search.minPrice")} />
        <FormField name="max_price" type="number" label={t("pages.search.maxPrice")} />

        <div className="flex flex-col gap-1.5">
          <label className="text-muted" htmlFor="sort">
            {t("pages.search.sort")}
          </label>
          <select
            id="sort"
            className="focus:shadow-outline border-line bg-surface text-foreground w-full rounded border px-3 py-2 leading-tight shadow focus:outline-none"
            {...form.register("sort")}
          >
            {SORTS.map((option) => (
              <option key={option} value={option}>
                {t(`pages.search.sortOptions.${option}`)}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex flex-wrap gap-3">
        <Button type="submit">{t("pages.search.apply")}</Button>
        <Button variant="secondary" onClick={handleClear}>
          {t("pages.search.clearAll")}
        </Button>
      </div>
    </Form>
  );
}
