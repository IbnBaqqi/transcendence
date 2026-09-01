import type { Document } from "../../components/types/DocumentTypes";

export const terms: Document = {
  title: "Käyttöehdot",
  lastUpdated: "Heinäkuu 2026",
  intro: [
    "Rakennettu 42-projektina Hivessä, Helsingissä. Ei kaupallinen lakipalvelu.",
    "Nämä ehdot voivat muuttua ajan myötä.",
  ],
  blocks: [
    {
      title: "Tilin vastuut",
      paragraphs: ["Pidä ilmoituksesi tarkkoina ja noudata alueesi sienestys- ja marjastuslakeja."],
    },
    {
      title: "Metsästäminen ja elintarviketurvallisuus",
      paragraphs: [
        "Luonnonruokaan liittyy aina riskejä. Sienten tai kasvien väärä tunnistaminen voi olla vaarallista.",
      ],
      bullets: [
        "Myyjät vastaavat oikeasta tunnistamisesta.",
        "Ostajat ottavat ostamiensa tuotteiden riskin.",
        "Alusta ei varmenna mitään tuotteita.",
      ],
    },
    {
      title: "Maksut ja tilaukset",
      paragraphs: [
        "Kaikki tämän alustan maksut ja tilaukset ovat simuloituja, eikä oikeita kauppoja tapahdu.",
      ],
    },
    {
      title: "Sallittu käyttö",
      paragraphs: [
        "Ei häirintää, petoksia tai laittomia ilmoituksia. Tätä rikkovat tilit voidaan keskeyttää.",
      ],
    },
    {
      title: "Vastuuvapauslausekkeet",
      paragraphs: ['Tarjotaan "sellaisenaan" ilman takuuta. Käytä omalla vastuullasi.'],
    },
  ],
  contactEmail: "support@metsatori.fi",
};
