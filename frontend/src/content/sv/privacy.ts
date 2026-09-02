import type { Document } from "../../components/types/DocumentTypes";

export const privacy: Document = {
  title: "Integritetspolicy",
  lastUpdated: "Juli 2026",
  intro: ["Byggt som ett 42-projekt vid Hive, Helsingfors. Inte en kommersiell juridisk tjänst."],
  blocks: [
    {
      title: "Vad vi samlar in",
      paragraphs: ["Vi kan samla in teknisk information som behövs för att driva tjänsten."],
      bullets: [
        "Kontouppgifter: e-post, visningsnamn, valfri region/profilbild",
        "Innehåll du skapar: annonser, meddelanden, recensioner",
        "Teknisk data: autentiseringstoken lagras i din webbläsare, inga annonsspårare",
      ],
    },
    {
      title: "Hur det används & vem som ser det",
      paragraphs: [
        "Vi använder din information för att tillhandahålla och förbättra tjänsten, upprätthålla säkerhet och kommunicera viktiga uppdateringar.",
        "Andra användare kan se din profil, dina annonser och meddelanden du skickar till dem.",
      ],
    },
    {
      title: "Lagring & säkerhet",
      paragraphs: [
        "Vi behåller information endast så länge som nödvändigt. Du kan när som helst begära åtkomst, rättelse eller radering av dina uppgifter. Alla lösenord är hashade.",
      ],
    },
    {
      title: "Betalningar",
      paragraphs: [
        "Betalningar på denna plattform är simulerade, ingen verklig finansiell data samlas in.",
      ],
    },
  ],
  contactEmail: "support@metsatori.fi",
};
