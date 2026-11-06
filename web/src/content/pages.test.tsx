import { render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ContentPage } from '../components/Prose';
import { I18nProvider } from '../i18n/i18n';
import { about, privacy, type PageContent, type Part } from './pages';

const text = (c: PageContent) =>
  [c.title, c.lead, ...c.sections.flatMap((s) => [s.heading, ...s.paragraphs.flatMap((p) => p.map((x: Part) => (typeof x === 'string' ? x : x.text)))])].join(' ');
const links = (c: PageContent) => c.sections.flatMap((s) => s.paragraphs.flatMap((p) => p.filter((x): x is { text: string; href: string } => typeof x !== 'string').map((l) => l.href)));

describe.each([
  ['privacy', privacy],
  ['about', about],
])('%s content', (_, page) => {
  it('has the same sections and links in both languages', () => {
    expect(page.en.sections).toHaveLength(page.nl.sections.length);
    page.nl.sections.forEach((s, i) => expect(page.en.sections[i]!.paragraphs).toHaveLength(s.paragraphs.length));
    expect(links(page.en)).toEqual(links(page.nl));
  });

  it('links only to https sites, and has no em dashes', () => {
    for (const l of [...links(page.nl), ...links(page.en)]) expect(l).toMatch(/^https:\/\//);
    expect(text(page.nl) + text(page.en)).not.toContain('\u2014');
  });
});

describe('privacy statement', () => {
  // The commitments the code keeps (backend/internal/api, src/flow): the text must state them.
  it.each([
    ['nl', ['niet opgeslagen', 'niet gelogd', 'geen cookies', 'Cloudflare', 'IP-adres', 'Pim van Leeuwen', 'Autoriteit Persoonsgegevens', 'nooit in het webadres']],
    ['en', ['not stored', 'not logged', 'no cookies', 'Cloudflare', 'IP address', 'Pim van Leeuwen', 'Autoriteit Persoonsgegevens', 'never in the web address']],
  ] as const)('%s states what is and is not kept', (lang, facts) => {
    const all = text(privacy[lang]).toLowerCase();
    for (const f of facts) expect(all, f).toContain(f.toLowerCase());
  });
});

describe('about page', () => {
  it('attributes the data to NS and says it is not affiliated', () => {
    expect(text(about.nl)).toContain('niet verbonden aan NS');
    expect(text(about.en)).toContain('not affiliated with or endorsed by NS');
    expect(links(about.nl)).toContain('https://apiportal.ns.nl');
  });
});

describe('ContentPage', () => {
  beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl']));
  afterEach(() => vi.restoreAllMocks());

  it('renders sections, and opens links in a new tab without a referrer', () => {
    render(
      <I18nProvider>
        <ContentPage content={privacy.nl} />
      </I18nProvider>,
    );
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Privacy');
    expect(screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)).toEqual(privacy.nl.sections.map((s) => s.heading));
    const cf = screen.getByRole('link', { name: /privacyverklaring van Cloudflare/ });
    expect(cf).toHaveAttribute('target', '_blank');
    expect(cf).toHaveAttribute('rel', 'noopener noreferrer');
    expect(within(cf).getByText('(opent in een nieuw tabblad)')).toBeInTheDocument();
    expect(screen.getByText('Laatst bijgewerkt: 6 november 2026')).toBeInTheDocument();
  });
});
