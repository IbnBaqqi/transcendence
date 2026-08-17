import { z } from "zod";
import { citySchema } from "./common";

export const addListingSchema = z.object({
  title: z.string().min(1, "Title is required"),
  description: z.string().min(1, "Description is required"),
  city: citySchema,
});

export type AddListingFormValues = z.infer<typeof addListingSchema>;
