// The privacy statement and the about page, in Dutch and English. Every claim here describes what
// the code does (backend/internal/api, src/flow); change them together. A paragraph is a list of
// parts: plain text, or a link.
import type { Lang } from '../i18n/i18n';

export type Part = string | { text: string; href: string };
export type Section = { heading: string; paragraphs: Part[][] };
export type PageContent = { title: string; lead: string; sections: Section[]; updated: string };

export const repoUrl = 'https://github.com/PimVanLeeuwen/dezetrein';
export const issuesUrl = `${repoUrl}/issues`;
const matchingUrl = `${repoUrl}/blob/main/docs/matching.md`;
const cloudflarePrivacy = 'https://www.cloudflare.com/privacypolicy/';
const nsApi = 'https://apiportal.ns.nl';
const ap = 'https://www.autoriteitpersoonsgegevens.nl';

export const privacy: Record<Lang, PageContent> = {
  nl: {
    title: 'Privacy',
    lead: 'dezetrein bewaart niets over jou. Geen account, geen cookies, geen analytics en geen trackers. Je locatie wordt alleen gebruikt om je trein te vinden.',
    updated: 'Laatst bijgewerkt: 6 november 2026',
    sections: [
      {
        heading: 'Je locatie',
        paragraphs: [
          ['Je browser vraagt pas om je locatie als je op "Deel mijn locatie" of "Gebruik mijn locatie" tikt. Je telefoon geeft dan een paar metingen: je positie, hoe nauwkeurig die is, en soms je snelheid en richting.'],
          ['Die metingen gaan in de inhoud van een verzoek naar de server van dezetrein, nooit in het webadres. Daar worden ze in het geheugen vergeleken met de posities van alle treinen, en meteen daarna weggegooid. Ze worden niet opgeslagen, niet gelogd en niet aan NS of iemand anders doorgegeven. Ook je browser bewaart ze niet.'],
        ],
      },
      {
        heading: 'Treinstel- en ritnummers',
        paragraphs: [['Een treinstel- of ritnummer dat je intypt gaat naar de server om de trein op te zoeken. Het wordt niet aan jou gekoppeld en niet bewaard.']],
      },
      {
        heading: 'Wat de server bijhoudt',
        paragraphs: [
          ['Per verzoek schrijft de server één logregel: wat voor soort verzoek het was (bijvoorbeeld "trein opvragen"), of het lukte en hoe lang het duurde. Niet je IP-adres, niet welk treinnummer en niet je locatie.'],
          ['Om misbruik te beperken telt de server in zijn geheugen hoeveel verzoeken er van een IP-adres komen. Die tellers worden nergens opgeslagen en verdwijnen na een paar minuten zonder verzoeken.'],
        ],
      },
      {
        heading: 'In je browser',
        paragraphs: [['dezetrein zet geen cookies. Alleen je keuze voor taal en thema wordt bewaard, in de lokale opslag van je browser, en verlaat je apparaat niet. Lettertypes en alle andere onderdelen komen van dezetrein zelf: je browser maakt geen verbinding met andere partijen, behalve als je op een link naar een andere site tikt, zoals ns.nl.']],
      },
      {
        heading: 'Hosting',
        paragraphs: [
          ['De site draait op een server in Nederland en is bereikbaar via Cloudflare. Cloudflare verwerkt verbindingsgegevens, zoals je IP-adres, om de site af te leveren en te beschermen. Lees daarover de ', { text: 'privacyverklaring van Cloudflare', href: cloudflarePrivacy }, '.'],
        ],
      },
      {
        heading: 'NS',
        paragraphs: [['De treininformatie komt van de ', { text: 'NS API', href: nsApi }, '. De server vraagt NS om treinposities en reisinformatie, en stuurt daarbij nooit gegevens over jou mee.']],
      },
      {
        heading: 'Vragen',
        paragraphs: [
          ['dezetrein wordt gemaakt door Pim van Leeuwen. Heb je een vraag of verzoek over privacy, open dan een issue op ', { text: 'GitHub', href: issuesUrl }, '. Omdat dezetrein geen gegevens over jou bewaart, is er niets in te zien, te wijzigen of te verwijderen.'],
          ['Ben je het ergens niet mee eens, dan kun je een klacht indienen bij de ', { text: 'Autoriteit Persoonsgegevens', href: ap }, '.'],
        ],
      },
    ],
  },
  en: {
    title: 'Privacy',
    lead: 'dezetrein keeps nothing about you. No account, no cookies, no analytics and no trackers. Your location is only used to find your train.',
    updated: 'Last updated: 6 November 2026',
    sections: [
      {
        heading: 'Your location',
        paragraphs: [
          ['Your browser only asks for your location when you tap "Share my location" or "Use my location". Your phone then gives a few readings: your position, how accurate it is, and sometimes your speed and direction.'],
          ['Those readings go to the dezetrein server in the body of a request, never in the web address. There they are compared in memory with the positions of all trains, and thrown away right after. They are not stored, not logged and not passed on to NS or anyone else. Your browser does not keep them either.'],
        ],
      },
      {
        heading: 'Trainset and journey numbers',
        paragraphs: [['A trainset or journey number you type goes to the server to look up the train. It is not linked to you and not stored.']],
      },
      {
        heading: 'What the server keeps',
        paragraphs: [
          ['For each request the server writes one log line: what kind of request it was (for example "look up a train"), whether it worked and how long it took. Not your IP address, not which train number and not your location.'],
          ['To limit abuse, the server counts in its memory how many requests come from an IP address. Those counters are never stored and disappear after a few minutes without requests.'],
        ],
      },
      {
        heading: 'In your browser',
        paragraphs: [['dezetrein sets no cookies. Only your choice of language and theme is kept, in your browser\'s local storage, and it does not leave your device. Fonts and everything else come from dezetrein itself: your browser connects to no other party, unless you tap a link to another site, such as ns.nl.']],
      },
      {
        heading: 'Hosting',
        paragraphs: [
          ['The site runs on a server in the Netherlands and is reached through Cloudflare. Cloudflare processes connection data, such as your IP address, to deliver and protect the site. See ', { text: "Cloudflare's privacy policy", href: cloudflarePrivacy }, '.'],
        ],
      },
      {
        heading: 'NS',
        paragraphs: [['Train information comes from the ', { text: 'NS API', href: nsApi }, '. The server asks NS for train positions and trip information, and never sends any data about you along.']],
      },
      {
        heading: 'Questions',
        paragraphs: [
          ['dezetrein is made by Pim van Leeuwen. For a question or request about privacy, open an issue on ', { text: 'GitHub', href: issuesUrl }, '. As dezetrein keeps no data about you, there is nothing to see, change or delete.'],
          ['If you disagree with something, you can file a complaint with the Dutch data protection authority, the ', { text: 'Autoriteit Persoonsgegevens', href: ap }, '.'],
        ],
      },
    ],
  },
};

export const about: Record<Lang, PageContent> = {
  nl: {
    title: 'Over dezetrein',
    lead: 'dezetrein beantwoordt één vraag: in welke trein zit ik, en hoe gaat het met de reis?',
    updated: 'Laatst bijgewerkt: 6 november 2026',
    sections: [
      {
        heading: 'Hoe het werkt',
        paragraphs: [
          ['Je telefoon geeft een paar metingen van je locatie, ook als die grof is. De server vergelijkt die met de live posities van alle NS-treinen, die hij elke 10 seconden één keer voor iedereen ophaalt. Omdat die posities altijd een beetje achterlopen, schat hij eerst waar elke trein nu echt is. Daarna vraagt de site je om te bevestigen: "Zit je in de trein naar Venlo?"'],
          ['Weet je het treinstelnummer, dat in de trein staat, of het ritnummer? Dan kun je ook daarmee zoeken, of je kiest je trein uit de vertrektijden van het station bij je in de buurt. ', { text: 'Lees hoe het zoeken werkt', href: matchingUrl }, ', en hoe goed het werkt op echte ritten.'],
        ],
      },
      {
        heading: 'Waar de gegevens vandaan komen',
        paragraphs: [
          ['Alle treininformatie komt van de ', { text: 'NS API', href: nsApi }, '. dezetrein laat zien wat NS publiceert, en zegt erbij hoe oud de gegevens zijn. Soms loopt dat achter of wijkt het af. Volg bij twijfel de borden en omroepen op het station en in de trein.'],
        ],
      },
      {
        heading: 'Niet van NS',
        paragraphs: [['dezetrein is een onafhankelijk project. Het is niet verbonden aan NS (Nederlandse Spoorwegen) en niet door NS goedgekeurd.']],
      },
      {
        heading: 'Open source',
        paragraphs: [
          ['De code is openbaar op ', { text: 'GitHub', href: repoUrl }, ', onder de GPL-3.0-licentie. De lettertypes zijn Bricolage Grotesque en Atkinson Hyperlegible, beide onder de SIL Open Font License.'],
        ],
      },
      {
        heading: 'Gemaakt door',
        paragraphs: [['Pim van Leeuwen. Ideeën, fouten of vragen zijn welkom als issue op ', { text: 'GitHub', href: issuesUrl }, '.']],
      },
    ],
  },
  en: {
    title: 'About dezetrein',
    lead: 'dezetrein answers one question: which train am I on, and how is the trip going?',
    updated: 'Last updated: 6 November 2026',
    sections: [
      {
        heading: 'How it works',
        paragraphs: [
          ['Your phone gives a few readings of your location, even a rough one. The server compares them with the live positions of all NS trains, which it fetches once every 10 seconds for everyone. As those positions always lag a little, it first estimates where each train really is now. Then the site asks you to confirm: "Are you on the train to Venlo?"'],
          ['Know the trainset number, shown inside the train, or the journey number? You can search with those too, or pick your train from the departures at a station near you. ', { text: 'Read how the search works', href: matchingUrl }, ', and how well it does on real trips.'],
        ],
      },
      {
        heading: 'Where the data comes from',
        paragraphs: [
          ['All train information comes from the ', { text: 'NS API', href: nsApi }, '. dezetrein shows what NS publishes, and says how old the data is. Sometimes it lags or differs. When in doubt, follow the signs and announcements at the station and on the train.'],
        ],
      },
      {
        heading: 'Not by NS',
        paragraphs: [['dezetrein is an independent project. It is not affiliated with or endorsed by NS (Nederlandse Spoorwegen).']],
      },
      {
        heading: 'Open source',
        paragraphs: [['The code is public on ', { text: 'GitHub', href: repoUrl }, ', under the GPL-3.0 licence. The fonts are Bricolage Grotesque and Atkinson Hyperlegible, both under the SIL Open Font License.']],
      },
      {
        heading: 'Made by',
        paragraphs: [['Pim van Leeuwen. Ideas, bugs or questions are welcome as an issue on ', { text: 'GitHub', href: issuesUrl }, '.']],
      },
    ],
  },
};
