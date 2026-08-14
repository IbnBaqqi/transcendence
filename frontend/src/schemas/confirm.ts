import { z } from "zod";
import { passwordSchema } from "./common";

export const confirmSchema = z
  .object({
    currentPassword: z.string().min(1, "Current password is required"),
    confirmPassword: z.string().min(1, "Password confirmation is required"),
  })
  .refine((data) => data.currentPassword === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  });

export type ConfirmFormValues = z.infer<typeof confirmSchema>;
