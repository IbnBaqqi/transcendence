// Zod schemas that we use to validate data closing #47 and for the user management module

import { z } from "zod";

import i18next from "../i18n";

// A message, not a string: resolved when the value is parsed rather than when
// this module loads, so it follows a language change. Without custom messages
// zod falls back to its own locale copy, which says things like "Invalid
// string: must match pattern /^\S+$/" - a regex, shown to a person.
const msg = (key: string) => ({ error: () => i18next.t(key) });

export const usernameSchema = z
  .string()
  .trim()
  .min(1, msg("validation.usernameRequired"))
  .max(50, msg("validation.usernameTooLong"))
  .regex(/^\S+$/, msg("validation.usernameNoSpaces"));
// NOTE: Backend all chars allowed, length 1-50 counted in runes, whitespace
// trimmed at edges, allowed inside, uniqueness case insensitive
export const emailSchema = z
  .string()
  .trim()
  .min(1, msg("validation.emailRequired"))
  .max(150, msg("validation.emailTooLong"))
  .email(msg("validation.emailInvalid"));
// NOTE: Backend caps firstname/lastname at 150 bytes (len() in
// validateProfileInput); the refine below enforces that stricter byte limit
// for multibyte text where the rune-counted max() alone wouldn't catch it.
export const nameSchema = z
  .string()
  .trim()
  .min(1, msg("validation.nameRequired"))
  .max(150, msg("validation.nameTooLong"))
  .refine((v) => new TextEncoder().encode(v).length <= 150, {
    params: { i18n: "validation.nameTooLong" },
  });
export const passwordSchema = z
  .string()
  .min(8, msg("validation.passwordTooShort"))
  .max(64, msg("validation.passwordTooLongChars"))
  .regex(/^\S+$/, msg("validation.passwordNoSpaces"));
// NOTE: Backend all chars allowed, length 8-72 counted in bytes, whitespace
// never trimmed

// A password being checked against a stored one, not created. The policy above
// is register's: applying it here rejects passwords that predate it, and tells
// the user their own password is wrong before anything has been asked.
export const existingPasswordSchema = z.string().min(1, msg("validation.passwordRequired"));
export const phoneSchema = z
  .string()
  .trim()
  .min(1, msg("validation.phoneRequired"))
  .max(15, msg("validation.phoneTooLong"))
  .regex(/^[\d\s()+-]{7,15}$/, msg("validation.phoneInvalid"));
// NOTE: If we want better validation then we could convert to E164 standard
// NOTE: Go rejects titles over 100 bytes (len() in validateListingInput) and
// the DB column is VARCHAR(100) which counts characters. The byte cap is the
// stricter one for multibyte text, so the refine below enforces it too.
export const titleSchema = z
  .string()
  .trim()
  .min(1, msg("validation.titleRequired"))
  .max(64, msg("validation.titleTooLong"))
  .refine((t) => new TextEncoder().encode(t).length <= 100, {
    params: { i18n: "validation.titleTooLongBytes" },
  });
// NOTE: Backend caps description at 1024 runes (maxDescriptionLength in
// service/listing.go); this mirrors it.
export const descriptionSchema = z.string().trim().max(1024, msg("validation.descriptionTooLong"));
export const categorySchema = z.string().trim().min(1, msg("validation.categoryRequired"));
export const priceSchema = z.number().positive(msg("validation.priceInvalid"));
export const quantitySchema = z.int32().positive(msg("validation.quantityInvalid"));
export const unitSchema = z
  .string()
  .trim()
  .min(1, msg("validation.unitRequired"))
  .max(20, msg("validation.unitTooLong"));

// NOTE: Backend caps location at 100 bytes (len() in validateProfileInput,
// shared with the addresses table); the refine below enforces that stricter
// byte limit for multibyte text where the rune-counted max() alone wouldn't.
export const locationSchema = z
  .string()
  .trim()
  .min(1, msg("validation.locationRequired"))
  .max(100, msg("validation.locationTooLong"))
  .regex(/^[\p{L}\s.'-]+$/u, msg("validation.locationInvalidChars"))
  .refine((v) => new TextEncoder().encode(v).length <= 100, {
    params: { i18n: "validation.locationTooLong" },
  });
// NOTE: If we want real geodata then we will need to link to an API such as OpenMaps

// NOTE: Exports for common schemas that are directly used without wrapper objects
export const bioSchema = z.object({
  bio: z.string().trim().max(1000, msg("validation.bioTooLong")),
});
export type BioFormValues = z.infer<typeof bioSchema>;
