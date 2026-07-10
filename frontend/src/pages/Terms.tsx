import { Terms } from "../content/Terms.ts";
import { LegalDocument } from "../components/text/LegalText.tsx";

export default function TermsOfService() {
  return <LegalDocument document={Terms} />;
}
