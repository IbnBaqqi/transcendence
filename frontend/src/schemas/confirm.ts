import { z } from "zod";

export const confirmSchema = z
  .object({
    password: z.string().min(1),
    confirmPassword: z.string().min(1),
  })
  .refine((data) => data.password === data.confirmPassword, {
    params: { i18n: "validation.passwordsMismatch" },
    path: ["confirmPassword"],
  });

export type ConfirmFormValues = z.infer<typeof confirmSchema>;
