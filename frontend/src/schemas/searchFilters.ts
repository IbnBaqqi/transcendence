import { z } from "zod";

import i18next from "../i18n";
import { SORTS } from "../lib/searchFilters";

const msg = (key: string) => ({ error: () => i18next.t(key) });

// An empty type="number" input arrives as NaN (FormField registers with
// valueAsNumber) and z.number() rejects it, so NaN is this form's "unset".
const optionalPrice = z.number().nonnegative(msg("validation.priceNegative")).or(z.nan());

export const searchFiltersSchema = z
  .object({
    keyword: z.string().trim(),
    category: z.string(),
    location: z.string().trim(),
    min_price: optionalPrice,
    max_price: optionalPrice,
    sort: z.enum(SORTS),
  })
  // Mirrors the backend's "Min price must not exceed max_price".
  .refine(
    (d) => Number.isNaN(d.min_price) || Number.isNaN(d.max_price) || d.min_price <= d.max_price,
    { params: { i18n: "validation.priceRangeInvalid" }, path: ["max_price"] },
  );

export type SearchFiltersFormValues = z.infer<typeof searchFiltersSchema>;
