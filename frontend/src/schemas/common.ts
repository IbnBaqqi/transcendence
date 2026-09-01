// Zod schemas that we use to validate data closing #47 and for the user management module

import { z } from "zod";

export const usernameSchema = z.string().trim().min(1).max(50).regex(/^\S+$/);
// NOTE: Backend all chars allowed, length 1-50 counted in runes, whitespace
// trimmed at edges, allowed inside, uniqueness case insensitive
export const emailSchema = z.string().trim().min(1).max(150).email();
// NOTE: Backend caps firstname/lastname at 150 bytes (len() in
// validateProfileInput); the refine below enforces that stricter byte limit
// for multibyte text where the rune-counted max() alone wouldn't catch it.
export const nameSchema = z
  .string()
  .trim()
  .min(1)
  .max(150)
  .refine((v) => new TextEncoder().encode(v).length <= 150, {
    params: { i18n: "validation.nameTooLong" },
  });
export const passwordSchema = z.string().min(8).max(64).regex(/^\S+$/);
// NOTE: Backend all chars allowed, length 8-72 counted in bytes, whitespace
// never trimmed
export const phoneSchema = z
  .string()
  .trim()
  .min(1)
  .max(15)
  .regex(/^[\d\s()+-]{7,15}$/);
// NOTE: If we want better validation then we could convert to E164 standard
// NOTE: Go rejects titles over 100 bytes (len() in validateListingInput) and
// the DB column is VARCHAR(100) which counts characters. The byte cap is the
// stricter one for multibyte text, so the refine below enforces it too.
export const titleSchema = z
  .string()
  .trim()
  .min(1)
  .max(64)
  .refine((t) => new TextEncoder().encode(t).length <= 100, {
    params: { i18n: "validation.titleTooLongBytes" },
  });
// NOTE: Backend imposes no length limit on description; 1024 here is a UI-only cap
export const descriptionSchema = z.string().trim().max(1024);
export const categorySchema = z.string().trim().min(1);
export const priceSchema = z.number().positive();
export const quantitySchema = z.int32().positive();
export const unitSchema = z.string().trim().min(1).max(20);

// NOTE: Backend caps location at 100 bytes (len() in validateProfileInput,
// shared with the addresses table); the refine below enforces that stricter
// byte limit for multibyte text where the rune-counted max() alone wouldn't.
export const locationSchema = z
  .string()
  .trim()
  .min(1)
  .max(100)
  .regex(/^[\p{L}\s.'-]+$/u)
  .refine((v) => new TextEncoder().encode(v).length <= 100, {
    params: { i18n: "validation.locationTooLong" },
  });
// NOTE: If we want real geodata then we will need to link to an API such as OpenMaps

// NOTE: Exports for common schemas that are directly used without wrapper objects
export const bioSchema = z.object({
  bio: z.string().trim().max(1000),
});
export type BioFormValues = z.infer<typeof bioSchema>;
