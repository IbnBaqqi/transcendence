import { makeAddListingSchema } from "../schemas/addListing";
import i18next from "../i18n";

const BASE = { title: "T", description: "", price: 1, quantity: 1, unit: "kg" };

describe("zod i18n", () => {
  beforeEach(async () => {
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("built-in checks use the English locale by default", () => {
    const result = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "" });

    expect(result.error?.issues[0].message).toBe(
      "Too small: expected string to have >=1 characters",
    );
  });

  test("custom refine checks resolve through translation keys", () => {
    const result = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "truffles" });

    expect(result.error?.issues[0].message).toBe("Choose a category from the list");
  });

  test("switching language reconfigures built-in and custom messages alike", async () => {
    await i18next.changeLanguage("fi");

    const empty = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "" });
    expect(empty.error?.issues[0].message).toBe("Liian pieni: merkkijonon täytyy olla >=1 merkkiä");

    const invalid = makeAddListingSchema(["mushrooms"]).safeParse({
      ...BASE,
      category: "truffles",
    });
    expect(invalid.error?.issues[0].message).toBe("Valitse luokka luettelosta");
  });
});
