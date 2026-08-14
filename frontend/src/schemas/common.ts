// Zod schemas that we use to validate data closing #47 and for the user management module

import { z } from "zod";

export const emailSchema = z.string().min(1, "Email is required").email("Invalid email address");
export const nameSchema = z
  .string()
  .min(1, "Name is required")
  .max(64, "Name must be less than 64 characters");
export const passwordSchema = z
  .string()
  .min(8, "Password must be at least 8 characters long")
  .max(64, "Password must be less than 64 characters");
export const phoneSchema = z
  .string()
  .min(1, "Phone number is required")
  .regex(/^[\d\s()+-]{7,20}$/, "Invalid phone number");
// NOTE: If we want better validation then we could convert to E164 standard
export const citySchema = z
  .string()
  .min(1, "City is required")
  .max(64, "City name is too long")
  .regex(/^[\p{L}\s.'-]+$/u, "Invalid city name");
// NOTE: If we want real geodata then we will need to link to an API such as OpenMaps

// TODO: Add schemas for listing data

// NOTE: Exports for common schemas that are directly used without wrapper objects
export const bioSchema = z
  .string()
  .max(1024, "Bio must be less than 1024 characters");
export type BioFormValues = z.infer<typeof bioSchema>;
