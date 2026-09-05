import { z } from "zod";
import { existingPasswordSchema, passwordSchema } from "./common";

export const changePasswordSchema = z
  .object({
    currentPassword: existingPasswordSchema,
    newPassword: passwordSchema,
    confirmPassword: passwordSchema,
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    params: { i18n: "validation.passwordsMismatch" },
    path: ["confirmPassword"],
  });

export type ChangePasswordFormValues = z.infer<typeof changePasswordSchema>;
