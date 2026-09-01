import { makeAddListingSchema } from "../schemas/addListing";

const VALID = {
  title: "Golden Chanterelles",
  description: "",
  price: 18,
  quantity: 4,
  unit: "kg",
};

const SLUGS = ["mushrooms", "chanterelles"];

describe("makeAddListingSchema", () => {
  test("accepts a slug that is in the list", () => {
    const result = makeAddListingSchema(SLUGS).safeParse({ ...VALID, category: "chanterelles" });
    expect(result.success).toBe(true);
  });

  test("rejects a slug that is not in the list, however well formed", () => {
    const result = makeAddListingSchema(SLUGS).safeParse({ ...VALID, category: "truffles" });

    expect(result.success).toBe(false);
    expect(result.error?.issues[0].message).toBe("Choose a category from the list");
  });

  test("accepts a slug the old format check would have rejected", () => {
    const result = makeAddListingSchema(["chicken_of_the_woods"]).safeParse({
      ...VALID,
      category: "chicken_of_the_woods",
    });

    expect(result.success).toBe(true);
  });

  test("an empty category is required, not unrecognised", () => {
    const result = makeAddListingSchema(SLUGS).safeParse({ ...VALID, category: "" });

    expect(result.success).toBe(false);
    expect(result.error?.issues[0].message).toBe(
      "Too small: expected string to have >=1 characters",
    );
  });
});
