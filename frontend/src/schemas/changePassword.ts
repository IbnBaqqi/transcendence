import { z } from "zod";
import { passwordSchema } from "./common";

export const changePasswordSchema = z
  .object({
    currentPassword: z.string().min(1),
    newPassword: passwordSchema,
    confirmPassword: passwordSchema,
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    params: { i18n: "validation.passwordsMismatch" },
    path: ["confirmPassword"],
  });

export type ChangePasswordFormValues = z.infer<typeof changePasswordSchema>;
