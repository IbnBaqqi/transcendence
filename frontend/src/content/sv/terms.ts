import type { Document } from "../../components/types/DocumentTypes";

export const terms: Document = {
  title: "Användarvillkor",
  lastUpdated: "Juli 2026",
  intro: [
    "Byggt som ett 42-projekt vid Hive, Helsingfors. Inte en kommersiell juridisk tjänst.",
    "Dessa villkor kan ändras över tid.",
  ],
  blocks: [
    {
      title: "Kontots ansvar",
      paragraphs: ["Håll dina annonser korrekta och följ lokala plockningslagar i ditt område."],
    },
    {
      title: "Plockning & livsmedelssäkerhet",
      paragraphs: [
        "Plockad mat innebär verkliga risker. Att felidentifiera svampar eller växter kan vara farligt.",
      ],
      bullets: [
        "Säljare ansvarar för korrekt identifiering.",
        "Köpare tar risken för det de köper.",
        "Plattformen verifierar inga varor.",
      ],
    },
    {
      title: "Betalningar & beställningar",
      paragraphs: [
        "Alla betalningar och beställningar på denna plattform är simulerade, inga verkliga försäljningar sker.",
      ],
    },
    {
      title: "Acceptabel användning",
      paragraphs: [
        "Ingen trakassering, bedrägeri eller olagliga annonser. Konton som bryter mot detta kan stängas av.",
      ],
    },
    {
      title: "Friskrivningar & ansvar",
      paragraphs: ['Tillhandahålls "i befintligt skick", utan garanti. Använd på egen risk.'],
    },
  ],
  contactEmail: "support@metsatori.fi",
};
