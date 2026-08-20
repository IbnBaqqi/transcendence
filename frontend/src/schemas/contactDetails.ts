import { z } from "zod";
import { nameSchema, phoneSchema, locationSchema } from "./common";

export const contactDetailsSchema = z.object({
  firstname: nameSchema,
  lastname: nameSchema.optional().or(z.literal("")),
  phone_number: phoneSchema.optional().or(z.literal("")),
  location: locationSchema,
});

export type ContactDetailsFormValues = z.infer<typeof contactDetailsSchema>;
