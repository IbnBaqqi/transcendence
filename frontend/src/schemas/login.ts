import { z } from "zod";
import { emailSchema, existingPasswordSchema } from "./common";

export const loginSchema = z.object({
  email: emailSchema,
  password: existingPasswordSchema,
});

export type LoginFormSchema = z.infer<typeof loginSchema>;
