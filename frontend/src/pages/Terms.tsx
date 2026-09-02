import { useLegalDocument } from "../content";
import { LegalDocument } from "../components/text/LegalText.tsx";

export default function TermsOfService() {
  const document = useLegalDocument("terms");
  return <LegalDocument document={document} />;
}
