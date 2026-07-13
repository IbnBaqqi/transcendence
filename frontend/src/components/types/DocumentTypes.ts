export type Block = {
  title: string;
  paragraphs: string[];
  bullets?: string[];
};

export type Document = {
  title: string;
  lastUpdated: string;
  intro: string[];
  blocks: Block[];
  contactEmail?: string;
};
