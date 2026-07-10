import type { Document } from "../components/types/DocumentTypes.ts";

export const Terms: Document = {
  title: "Terms of Sevice",
  lastUpdated: "July 2026",
  intro: [
    "Built for a 42 curriculum project at Hive, Helsinki. Not a commercial legal service.",
    "These terms may change over time.",
  ],
  blocks: [
    {
      title: "Account responsibilities",
      paragraphs: [
        "Keep your listings accurate and follow local foraging laws in your area.",
      ],
    },
    {
      title: "Foraging & food safety",
      paragraphs: [
        "Foraged foods carry real risk. Misidentifying mushrooms or plants can be dangerous.",
      ],
      bullets: [
        "Sellers are responsible for correct identification.",
        "Buyers assume the risk of what they purchase.",
        "The platform does not verify any goods.",
      ],
    },
    {
      title: "Payments & orders",
      paragraphs: [
        "All payments and orders on this platform are simulated, no real sales take place.",
      ],
    },
    {
      title: "Acceptable use",
      paragraphs: [
        "No harassment, fraud, or illegal listings. Accounts violating this may be suspended.",
      ],
    },
    {
      title: "Disclaimers & liability",
      paragraphs: [
        "Provided \"as is,\" with no warranty. Use at your own risk.",
      ],
    },
  ],
  contactEmail: "support@forageapp.fi",
};
