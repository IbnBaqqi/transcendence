import i18next from "../i18n";
import {
  usernameSchema,
  existingPasswordSchema,
  passwordSchema,
  phoneSchema,
  emailSchema,
  locationSchema,
} from "../schemas/common";

type Parsable = {
  safeParse: (v: unknown) => { success: boolean; error?: { issues: { message: string }[] } };
};

// Every check with a message of its own, and a value that trips it.
const cases: [string, Parsable, unknown][] = [
  ["username: empty", usernameSchema, ""],
  ["username: spaces", usernameSchema, "two words"],
  ["username: too long", usernameSchema, "x".repeat(51)],
  ["password: too short", passwordSchema, "abc"],
  ["password: empty", existingPasswordSchema, ""],
  ["password: spaces", passwordSchema, "abc def ghij"],
  ["phone: bad shape", phoneSchema, "!!!"],
  ["email: bad shape", emailSchema, "nope"],
  ["location: digits", locationSchema, "Helsinki 00100"],
];

function messageFor(schema: Parsable, value: unknown): string {
  const result = schema.safeParse(value);
  if (result.success) throw new Error("expected this value to be rejected");
  return result.error!.issues[0].message;
}

describe("validation messages", () => {
  // Dropping the custom messages hands the user zod's own locale copy, which
  // reads "Invalid string: must match pattern /^\S+$/" - a regex, shown to a
  // person - and "Too small: expected string to have >=1 characters" for a
  // field that has a name. Both are localized, and both are unacceptable.
  test.each(["en", "fi", "sv"])("%s says something a person can act on", async (lng) => {
    await i18next.changeLanguage(lng);

    for (const [label, schema, value] of cases) {
      const message = messageFor(schema, value);
      expect(message, `${label} leaked a pattern`).not.toMatch(/\/\^|pattern|Invalid string/i);
      expect(message, `${label} used zod's generic phrasing`).not.toMatch(
        /too small|too big|liian pieni|för lite/i,
      );
      expect(message.length, `${label} produced an empty message`).toBeGreaterThan(0);
    }
  });

  test("the three locales do not share a single message", async () => {
    const seen = new Set<string>();
    for (const lng of ["en", "fi", "sv"]) {
      await i18next.changeLanguage(lng);
      seen.add(messageFor(usernameSchema, "two words"));
    }
    // Catches a missing key falling back to English in every locale.
    expect(seen.size).toBe(3);
  });
});
