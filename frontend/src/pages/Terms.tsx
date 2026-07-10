import { Terms } from "../content/Terms.ts";

export default function TermsOfService() {
  const { title, lastUpdated, intro, sections, contactEmail } = Terms;
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">Terms of Service</h1>
      <p className="text-muted mt-4">PLACEHOLDER</p>
    </div>
  );
}
