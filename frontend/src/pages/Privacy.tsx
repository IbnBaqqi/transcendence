import { Privacy } from "../content/Privacy.ts";
import { LegalDocument } from "../components/text/LegalText.tsx";

export default function PrivacyPolicy() {
  return <LegalDocument document={Privacy} />;
}
/*
export default function PrivacyPolicy() {
  const { title, lastUpdated, intro, blocks, contactEmail } = Privacy;

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{Privacy.title}</h1>
      <p className="text-muted mt-2 text-sm">Last updated: {Privacy.lastUpdated}</p>

      <div className="text-muted mt-6 space-y-4">
        {intro.map((paragraph) => (
          <p key={paragraph}>{paragraph}</p>
        ))}
      </div>

      <div className="mt-8 space-y-8">
        {blocks.map(({ title, paragraphs, bullets }) => (
          <section key={title}>
            <h2 className="text-foreground text-xl font-semibold">{title}</h2>

            <div className="text-muted mt-3 space-y-3">
              {paragraphs.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </div>

            {bullets?.length ? (
              <ul className="text-muted mt-3 list-disc space-y-1 pl-5">
                {bullets.map((bullet) => (
                  <li key={bullet}>{bullet}</li>
                ))}
              </ul>
            ) : null}
          </section>
        ))}
      </div>

      {contactEmail ? (
        <p className="text-muted mt-10">
          Questions? Reach out at:{" "}
          <a href={`mailto:${contactEmail}`} className="text-accent hover:underline">
            {contactEmail}
          </a>
        </p>
      ) : null}
    </div>
  );
}
*/
