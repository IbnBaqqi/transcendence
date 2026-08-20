import { z } from "zod";
import { usernameSchema, emailSchema, passwordSchema } from "./common";

export const signupSchema = z.object({
  username: usernameSchema,
  email: emailSchema,
  password: passwordSchema,
});

export type SignupFormValues = z.infer<typeof signupSchema>;
