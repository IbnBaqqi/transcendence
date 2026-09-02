import type { Document } from "../../components/types/DocumentTypes";

export const privacy: Document = {
  title: "Tietosuojakäytäntö",
  lastUpdated: "Heinäkuu 2026",
  intro: ["Rakennettu 42-projektina Hivessä, Helsingissä. Ei kaupallinen lakipalvelu."],
  blocks: [
    {
      title: "Mitä keräämme",
      paragraphs: ["Voimme kerätä palvelun toiminnan edellyttämää teknistä tietoa."],
      bullets: [
        "Tilitiedot: sähköposti, näyttönimi, valinnainen alue/profiilikuva",
        "Luomasi sisältö: ilmoitukset, viestit, arvostelut",
        "Tekniset tiedot: selaimeen tallennetut kirjautumistunnisteet, ei mainosseurantaa",
      ],
    },
    {
      title: "Miten tietoja käytetään ja kuka ne näkee",
      paragraphs: [
        "Käytämme tietojasi palvelun tarjoamiseen ja parantamiseen, tietoturvan ylläpitoon sekä tärkeistä päivityksistä tiedottamiseen.",
        "Muut käyttäjät voivat nähdä profiilisi, ilmoituksesi ja lähettämäsi viestit.",
      ],
    },
    {
      title: "Säilytys ja turvallisuus",
      paragraphs: [
        "Säilytämme tietoja vain niin kauan kuin on tarpeellista. Voit milloin tahansa pyytää tietojesi tarkastusta, korjausta tai poistoa. Kaikki salasanat on tiivistetty.",
      ],
    },
    {
      title: "Maksut",
      paragraphs: ["Tämän alustan maksut ovat simuloituja, ja oikeita maksutietoja ei kerätä."],
    },
  ],
  contactEmail: "support@metsatori.fi",
};
