import { z } from "zod";

// Parameterised because the expected value is the signed-in user's own name,
// which the schema cannot know statically - same shape as makeAddListingSchema.
//
// Exact and case-sensitive: the server compares confirmation != user.Username
// with no normalising (service/user.go), so anything looser here would let a
// form pass that the API then rejects.
export function makeDeleteAccountSchema(username: string) {
  return z.object({
    confirmation: z.string().refine((value) => value === username, {
      params: { i18n: "validation.confirmUsernameMismatch" },
    }),
  });
}

export type DeleteAccountFormValues = z.infer<ReturnType<typeof makeDeleteAccountSchema>>;
