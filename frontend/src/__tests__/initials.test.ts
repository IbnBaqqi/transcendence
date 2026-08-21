import { deriveInitials } from "../lib/initials";

// The backend guarantees a signed-in user has a non-empty username, so there
// is no "?" branch here - that decision belongs to the call sites.
describe("deriveInitials", () => {
  test("prefers one initial from each real name", () => {
    expect(deriveInitials("or99", "Oscar", "Rogers")).toBe("OR");
  });

  test("trims and uppercases the names", () => {
    expect(deriveInitials("or99", " oscar ", " rogers ")).toBe("OR");
  });

  test("falls back to a single initial when only a first name exists", () => {
    expect(deriveInitials("or99", "Oscar", null)).toBe("O");
    expect(deriveInitials("or99", "Oscar", undefined)).toBe("O");
  });

  test("uses the username when no names are set", () => {
    expect(deriveInitials("or99", null, null)).toBe("O");
  });

  test("treats whitespace-only names as unset", () => {
    expect(deriveInitials("or99", "   ", "\t")).toBe("O");
  });

  test("ignores a last name without a first name", () => {
    expect(deriveInitials("or99", null, "Rogers")).toBe("O");
  });

  test("is multibyte safe", () => {
    expect(deriveInitials("émile", "Émile", "Zola")).toBe("ÉZ");
    expect(deriveInitials("émile")).toBe("É");
  });
});
