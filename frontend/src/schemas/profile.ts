import { z } from "zod";
import { emailSchema, passwordSchema, nameSchema, phoneSchema, citySchema, bioSchema } from "./common";
import { contactDetailsSchema } from "./contactDetails";

export const profileSchema = contactDetailsSchema.extend({
  email: emailSchema,
  bio: bioSchema.optional().or(z.literal("")),
});

export type ProfileFormValues = z.infer<typeof profileSchema>;
