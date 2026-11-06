import { act, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { detectLang, I18nProvider, translate, useI18n } from './i18n';
import { en, nl } from './messages';

afterEach(() => {
  window.localStorage.clear();
  vi.restoreAllMocks();
});

describe('detectLang', () => {
  it.each([
    [null, ['en-GB', 'nl'], 'en'],
    [null, ['nl-BE', 'en'], 'nl'],
    [null, ['de-DE', 'en-US'], 'en'],
    [null, ['fr', 'de'], 'nl'],
    [null, [], 'nl'],
    ['en', ['nl'], 'en'],
    ['nl', ['en'], 'nl'],
    ['xx', ['en'], 'en'],
  ] as const)('stored %s, browser %j: %s', (stored, browser, want) => expect(detectLang(stored, browser)).toBe(want));
});

describe('messages', () => {
  it('has every key in both languages, with the same placeholders', () => {
    expect(Object.keys(en).sort()).toEqual(Object.keys(nl).sort());
    const vars = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();
    for (const k of Object.keys(nl) as (keyof typeof nl)[]) expect(vars(en[k]), k).toEqual(vars(nl[k]));
  });

  it('fills placeholders and leaves unknown ones', () => {
    expect(translate('en', 'train.title', { train: 3522 })).toBe('Train 3522');
    expect(translate('nl', 'train.title')).toBe('Trein {train}');
  });

  it('contains no em dashes', () => {
    for (const s of [...Object.values(nl), ...Object.values(en)]) expect(s).not.toContain('\u2014');
  });
});

function Probe() {
  const { t, lang, setLang } = useI18n();
  return (
    <button type="button" onClick={() => setLang(lang === 'nl' ? 'en' : 'nl')}>
      {t('home.find')}
    </button>
  );
}

describe('I18nProvider', () => {
  it('switches language, sets the page language and remembers the choice', () => {
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl-NL']);
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );
    expect(screen.getByRole('button')).toHaveTextContent('Vind mijn trein');
    expect(document.documentElement.lang).toBe('nl');
    act(() => screen.getByRole('button').click());
    expect(screen.getByRole('button')).toHaveTextContent('Find my train');
    expect(document.documentElement.lang).toBe('en');
    expect(window.localStorage.getItem('dezetrein.lang')).toBe('en');
  });

  it('still works when storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['en']);
    render(
      <I18nProvider>
        <Probe />
      </I18nProvider>,
    );
    expect(screen.getByRole('button')).toHaveTextContent('Find my train');
    act(() => screen.getByRole('button').click());
    expect(screen.getByRole('button')).toHaveTextContent('Vind mijn trein');
  });
});
