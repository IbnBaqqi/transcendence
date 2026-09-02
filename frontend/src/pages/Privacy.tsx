import { useLegalDocument } from "../content";
import { LegalDocument } from "../components/text/LegalText.tsx";

export default function PrivacyPolicy() {
  const document = useLegalDocument("privacy");
  return <LegalDocument document={document} />;
}
