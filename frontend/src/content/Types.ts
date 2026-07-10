export type Section = {
  title: string;
  paragraphs: string[];
  bullets?: string[];
};

export type Document = {
  title: string;
  lastUpdated: string;
  intro: string[];
  sections: Section[];
  contactEmail?: string;
};
