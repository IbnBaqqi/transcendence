import { z } from "zod";

import { makeAddListingSchema } from "../schemas/addListing";
import i18next from "../i18n";

const BASE = { title: "T", description: "", price: 1, quantity: 1, unit: "kg" };

describe("zod i18n", () => {
  beforeEach(async () => {
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  // Every check the forms use now carries its own message, because zod's
  // locale copy names the constraint rather than the field - and for a regex
  // it prints the pattern. What zod's locale still covers is everything we did
  // not write a message for, which is why it stays configured: a local schema
  // stands in for that here.
  test("checks we wrote messages for name the field", () => {
    const result = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "" });

    expect(result.error?.issues[0].message).toBe("Category is required");
  });

  test("zod's own locale still covers what we left to it", () => {
    const result = z.string().safeParse(42);

    expect(result.error?.issues[0].message).toBe("Invalid input: expected string, received number");
  });

  test("custom refine checks resolve through translation keys", () => {
    const result = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "truffles" });

    expect(result.error?.issues[0].message).toBe("Choose a category from the list");
  });

  test("switching language reconfigures built-in and custom messages alike", async () => {
    await i18next.changeLanguage("fi");

    const empty = makeAddListingSchema(["mushrooms"]).safeParse({ ...BASE, category: "" });
    expect(empty.error?.issues[0].message).toBe("Kategoria vaaditaan");

    // zod's own locale switches with it, for the checks we did not name.
    expect(z.string().safeParse(42).error?.issues[0].message).toBe(
      "Virheellinen tyyppi: odotettiin string, oli number",
    );

    const invalid = makeAddListingSchema(["mushrooms"]).safeParse({
      ...BASE,
      category: "truffles",
    });
    expect(invalid.error?.issues[0].message).toBe("Valitse luokka luettelosta");
  });
});
