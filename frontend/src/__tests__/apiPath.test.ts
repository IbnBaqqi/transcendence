import { apiPath } from "../api/client";

describe("apiPath", () => {
  test("leaves an ordinary id untouched", () => {
    expect(apiPath`/users/${"01a04477-9d41-7aa2-a62e-5f5e937a35fc"}`).toBe(
      "/users/01a04477-9d41-7aa2-a62e-5f5e937a35fc",
    );
  });

  // React Router percent-decodes params, so a crafted URL hands the component
  // a literal "../me/profile". Interpolated raw, "/api/v1/users/../me/profile"
  // collapses to "/api/v1/me/profile" - a different endpoint entirely.
  test("encodes a traversal attempt into a single path segment", () => {
    expect(apiPath`/users/${"../me/profile"}`).toBe("/users/..%2Fme%2Fprofile");
  });

  test("encodes every value, not just the first", () => {
    expect(apiPath`/listings/${"a/b"}/images/${"c/d"}`).toBe("/listings/a%2Fb/images/c%2Fd");
  });

  test("handles a trailing literal with no value after it", () => {
    expect(apiPath`/listings/${"1"}/images`).toBe("/listings/1/images");
  });
});
