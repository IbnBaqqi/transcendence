import i18next from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";
import * as z from "zod";

import en from "./locales/en/translation.json";
import fi from "./locales/fi/translation.json";
import sv from "./locales/sv/translation.json";

export const resources = {
  en: { translation: en },
  fi: { translation: fi },
  sv: { translation: sv },
} as const;

export type Locale = keyof typeof resources;
export const supportedLngs: Locale[] = ["en", "fi", "sv"];

/**
 * Resolves an i18next key for a zod issue. `.refine(fn, { params: { i18n } })`
 * leaves no `message` on the issue, so the error-resolution chain falls
 * through to this global `customError`, which translates the key. Returning
 * `undefined` lets zod's built-in `localeError` (en/fi/sv) handle everything
 * else (email, min, max, regex, ...).
 */
function zodCustomError(issue: z.core.$ZodRawIssue): { message: string } | undefined {
  const params = (issue as { params?: Record<string, unknown> }).params;
  const key = params?.i18n as string | undefined;
  if (typeof key === "string") {
    return { message: i18next.t(key, params?.values as Record<string, unknown>) };
  }
  return undefined;
}

function applyZodLocale(lng: string) {
  const locale =
    (z.locales as Record<string, () => { localeError: z.core.$ZodErrorMap }>)[lng] ?? z.locales.en;
  z.config({
    customError: zodCustomError,
    localeError: locale().localeError,
  });
}

// Keep <html lang> in sync with the active locale so screen readers and
// browser translation pick up the right language (see index.html).
function applyDocumentLang(lng: string) {
  if (typeof document !== "undefined") {
    document.documentElement.lang = lng;
  }
}

function applyLocale(lng: string) {
  applyZodLocale(lng);
  applyDocumentLang(lng);
}

void i18next
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs,
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "i18nextLng",
    },
    interpolation: {
      escapeValue: false,
    },
  });

applyLocale(i18next.language || "en");
i18next.on("languageChanged", (lng) => applyLocale(lng));

export default i18next;
