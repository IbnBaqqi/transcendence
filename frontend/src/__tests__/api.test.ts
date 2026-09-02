import { AxiosError, AxiosHeaders } from "axios";

import { isApiError, toApiError } from "../api/client";
import { toQueryString } from "../api/listings";
import i18next from "../i18n";

// Whatever toApiError accepts, without exporting the wire type just for a test.
type WireError = Parameters<typeof toApiError>[0];
type WireBody = NonNullable<WireError["response"]>["data"];

// Builds the AxiosError the interceptor would receive. Omit the argument for a
// request that never reached the server.
function axiosError(response?: { status: number; data: unknown }): WireError {
  const error = new AxiosError("request failed") as WireError;

  if (response) {
    error.response = {
      status: response.status,
      statusText: "",
      data: response.data as WireBody,
      headers: new AxiosHeaders(),
      config: { headers: new AxiosHeaders() },
    };
  }
  return error;
}

describe("toApiError", () => {
  beforeEach(async () => {
    // The localized-fallback tests switch language; keep the rest deterministic.
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  afterAll(async () => {
    // Leave the shared singleton back on the default so later describes in
    // this file (and any other test file in the same worker) stay green.
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("uses the backend's message on a 4xx", () => {
    const err = toApiError(axiosError({ status: 404, data: { error: "listing not found" } }));

    expect(err.status).toBe(404);
    expect(err.message).toBe("listing not found");
  });

  test("passes details through when present", () => {
    const err = toApiError(
      axiosError({ status: 400, data: { error: "invalid", details: { title: "required" } } }),
    );

    expect(err.details).toEqual({ title: "required" });
  });

  test("falls back when the server answers without a usable body", () => {
    const err = toApiError(axiosError({ status: 500, data: "" }));

    expect(err.status).toBe(500);
    expect(err.message).toBe("Something went wrong. Please try again.");
  });

  // status 0 is what lets a component say "check your connection" instead of
  // blaming the server for something it never received.
  test("uses status 0 when the request never reached the server", () => {
    const err = toApiError(axiosError());

    expect(err.status).toBe(0);
    expect(err.message).toBe("Could not reach the server");
  });

  test("passes through a backend-provided message verbatim, even when localized", async () => {
    await i18next.changeLanguage("fi");

    const err = toApiError(axiosError({ status: 404, data: { error: "listing not found" } }));

    expect(err.message).toBe("listing not found");
  });

  test("localizes the synthesized fallbacks to the active language", async () => {
    await i18next.changeLanguage("fi");

    expect(toApiError(axiosError({ status: 500, data: "" })).message).toBe(
      "Jokin meni pieleen. Yritä uudelleen.",
    );
    expect(toApiError(axiosError()).message).toBe("Palvelimeen ei saada yhteyttä");
  });
});

describe("isApiError", () => {
  test("accepts a normalised error", () => {
    expect(isApiError({ status: 404, message: "nope" })).toBe(true);
  });

  test("rejects anything else", () => {
    for (const value of [null, undefined, "boom", 500, new Error("boom"), {}]) {
      expect(isApiError(value)).toBe(false);
    }
  });
});

describe("toQueryString", () => {
  // The cache key is this string, so the same filters written in a different
  // order must not create a second cache entry.
  test("is stable regardless of the order the params were written in", () => {
    expect(toQueryString({ keyword: "chanterelle", page: 2 })).toBe(
      toQueryString({ page: 2, keyword: "chanterelle" }),
    );
  });

  test("drops undefined and empty values", () => {
    expect(toQueryString({ keyword: "", category: undefined, page: 1 })).toBe("page=1");
  });

  test("serialises numbers and keeps the sort key", () => {
    expect(toQueryString({ min_price: 5, sort: "price_asc" })).toBe("min_price=5&sort=price_asc");
  });

  test("is empty when nothing is filtered", () => {
    expect(toQueryString({})).toBe("");
  });
});
