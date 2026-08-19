import { z } from "zod";
import { emailSchema, passwordSchema } from "./common";

export const signupSchema = z.object({
  email: emailSchema,
  password: passwordSchema,
});

export type SignupFormValues = z.infer<typeof signupSchema>;
