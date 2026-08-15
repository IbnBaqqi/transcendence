import { z } from "zod";
import { nameSchema, phoneSchema, citySchema } from "./common";

export const contactDetailsSchema = z.object({
  firstName: nameSchema,
  lastName: nameSchema.optional().or(z.literal("")),
  phone: phoneSchema.optional().or(z.literal("")),
  location: citySchema,
});

export type ContactDetailsFormValues = z.infer<typeof contactDetailsSchema>;
