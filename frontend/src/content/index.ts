import { useTranslation } from "react-i18next";

import type { Document } from "../components/types/DocumentTypes";

import { privacy as enPrivacy } from "./en/privacy";
import { terms as enTerms } from "./en/terms";
import { privacy as fiPrivacy } from "./fi/privacy";
import { terms as fiTerms } from "./fi/terms";
import { privacy as svPrivacy } from "./sv/privacy";
import { terms as svTerms } from "./sv/terms";

const documents = {
  en: { privacy: enPrivacy, terms: enTerms },
  fi: { privacy: fiPrivacy, terms: fiTerms },
  sv: { privacy: svPrivacy, terms: svTerms },
} as const;

export type LegalDocumentKind = "privacy" | "terms";

/**
 * Returns the localized legal document for the current language, falling back
 * to English (the default and fallback locale) when the active language has
 * no entries of its own. That fallback is defensive: `supportedLngs` (see
 * src/i18n) is exactly en/fi/sv and all three have documents, so it is
 * currently unreachable.
 */
export function useLegalDocument(kind: LegalDocumentKind): Document {
  const { i18n } = useTranslation();
  const lng = (i18n.language?.split("-")[0] ?? "en") as keyof typeof documents;
  const locale = documents[lng] ?? documents.en;
  return locale[kind];
}
