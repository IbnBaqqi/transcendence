import { z } from "zod";
import {
  titleSchema,
  descriptionSchema,
  categorySchema,
  priceSchema,
  quantitySchema,
  unitSchema,
} from "./common";

export function makeAddListingSchema(validSlugs: readonly string[]) {
  return z.object({
    title: titleSchema,
    description: descriptionSchema,
    category: categorySchema.refine((slug) => validSlugs.includes(slug), {
      message: "Choose a category from the list",
    }),
    price: priceSchema,
    quantity: quantitySchema,
    unit: unitSchema,
  });
}

export type AddListingFormValues = z.infer<ReturnType<typeof makeAddListingSchema>>;
