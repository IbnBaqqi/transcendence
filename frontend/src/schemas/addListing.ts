import { z } from "zod";
import {
  titleSchema,
  descriptionSchema,
  categorySchema,
  priceSchema,
  quantitySchema,
  unitSchema,
} from "./common";

export const addListingSchema = z.object({
  title: titleSchema,
  description: descriptionSchema,
  category: categorySchema,
  price: priceSchema,
  quantity: quantitySchema,
  unit: unitSchema,
});

export type AddListingFormValues = z.infer<typeof addListingSchema>;
