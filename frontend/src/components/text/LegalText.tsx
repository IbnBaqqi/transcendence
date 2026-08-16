import type { Block, Document } from "../types/DocumentTypes.ts";
import Mailto from "../objects/Mailto.tsx";

function LegalParagraph({ paragraphs }: { paragraphs: string[] }) {
  return (
    <div className="text-muted mt-3 space-y-3">
      {paragraphs.map((text) => (
        <p key={text}>{text}</p>
      ))}
    </div>
  );
}

function LegalBullets({ bullets }: { bullets?: string[] }) {
  if (!bullets?.length) return null;

  return (
    <ul className="text-muted mt-3 list-disc space-y-1 pl-5">
      {bullets.map((item) => (
        <li key={item}>{item}</li>
      ))}
    </ul>
  );
}

function LegalSection({ block }: { block: Block }) {
  return (
    <section>
      <h2 className="text-foreground text-xl font-semibold">{block.title}</h2>
      <LegalParagraph paragraphs={block.paragraphs} />
      <LegalBullets bullets={block.bullets} />
    </section>
  );
}

export function LegalDocument({ document }: { document: Document }) {
  const { title, lastUpdated, intro, blocks, contactEmail } = document;

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{title}</h1>
      <p className="text-muted mt-2 text-sm">Last updated: {lastUpdated}</p>

      <div className="text-muted mt-6 space-y-4">
        {intro.map((text) => (
          <p key={text}>{text}</p>
        ))}
      </div>

      <div className="mt-8 space-y-8">
        {blocks.map((block) => (
          <LegalSection key={block.title} block={block} />
        ))}
      </div>

      {contactEmail && (
        <p className="text-muted mt-10">
          Questions? Reach out at: <Mailto label={contactEmail} mailto={contactEmail} />
        </p>
      )}
    </div>
  );
}
