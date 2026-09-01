import { renderHook, waitFor } from "@testing-library/react";

import { useLegalDocument } from "../content";
import i18next from "../i18n";

describe("useLegalDocument", () => {
  beforeEach(async () => {
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("returns the English document by default", () => {
    const { result } = renderHook(() => useLegalDocument("privacy"));

    expect(result.current.title).toBe("Privacy Policy");
  });

  test("returns the localized document for the active language", async () => {
    const { result } = renderHook(() => useLegalDocument("privacy"));

    await i18next.changeLanguage("fi");
    await waitFor(() => expect(result.current.title).toBe("Tietosuojakäytäntö"));

    await i18next.changeLanguage("sv");
    await waitFor(() => expect(result.current.title).toBe("Integritetspolicy"));
  });

  test("serves terms per language", async () => {
    const { result } = renderHook(() => useLegalDocument("terms"));

    await i18next.changeLanguage("fi");
    await waitFor(() => expect(result.current.title).toBe("Käyttöehdot"));
  });

  test("always returns a document, even for a language that has no documents", async () => {
    const { result } = renderHook(() => useLegalDocument("terms"));

    // i18next rejects "de" outright (`supportedLngs` is exactly en/fi/sv), so
    // it can never become the active language. This asserts that the outcome
    // is still a document rather than a crash, guarding the fallback in
    // useLegalDocument against future refactors - not a claim that the
    // fallback branch itself executes.
    await i18next.changeLanguage("de");
    await waitFor(() => expect(result.current.title).toBe("Terms of Service"));
  });
});
