import type { Document } from "../components/types/DocumentTypes.ts";

export const Privacy: Document = {
  title: "Privacy Policy",
  lastUpdated: "July 2026",
  intro: ["Built for a 42 curriculum project at Hive, Helsinki. Not a commercial legal service."],
  blocks: [
    {
      title: "What we collect",
      paragraphs: ["We may collect data technical information needed to operate the service."],
      bullets: [
        "Account details: email, display name, optional region/avatar",
        "Content you create: listings, messages, reviews",
        "Technical data: auth tokens stored in your browser, no ad trackers",
      ],
    },
    {
      title: "How it's used & who sees it",
      paragraphs: [
        "We use your information to provide and improve the service, maintain security, and communicate important updates.",
        "Other users can see your profile, listings, and messages you send them.",
      ],
    },
    {
      title: "Storage & security",
      paragraphs: [
        "We retain information only as long as necessary. You may also request access, correction, or deletion of your data at any time. All passwords are hashed.",
      ],
    },
    {
      title: "Payments",
      paragraphs: ["Payments on this platform are simulated, no real financial data is collected."],
    },
  ],
  contactEmail: "support@metsatori.fi",
};
