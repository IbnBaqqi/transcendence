import { z } from "zod";
import { emailSchema, passwordSchema } from "./common";

export const loginSchema = z.object({
  identifier: emailSchema,
  password: passwordSchema,
});

export type LoginFormSchema = z.infer<typeof loginSchema>;
