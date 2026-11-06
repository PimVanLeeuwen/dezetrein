import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { messages, type MessageKey } from './messages';

export type Lang = 'nl' | 'en';

const storageKey = 'dezetrein.lang';

// detectLang picks the stored choice, else the first of the browser's languages that the site
// speaks, else Dutch.
export function detectLang(stored: string | null, browser: readonly string[]): Lang {
  if (stored === 'nl' || stored === 'en') return stored;
  for (const l of browser) {
    const base = l.toLowerCase().split('-')[0];
    if (base === 'nl' || base === 'en') return base;
  }
  return 'nl';
}

export function translate(lang: Lang, key: MessageKey, vars?: Record<string, string | number>): string {
  const text: string = messages[lang][key];
  if (!vars) return text;
  return text.replace(/\{(\w+)\}/g, (m, name: string) => (name in vars ? String(vars[name]) : m));
}

function readStored(): string | null {
  try {
    return window.localStorage.getItem(storageKey);
  } catch {
    return null; // storage blocked (private mode): fall back to the browser language
  }
}

type I18n = {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: MessageKey, vars?: Record<string, string | number>) => string;
};

const Ctx = createContext<I18n | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => detectLang(readStored(), navigator.languages ?? [navigator.language]));
  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);
  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    try {
      window.localStorage.setItem(storageKey, l);
    } catch {
      // not remembered, but still switched
    }
  }, []);
  const value = useMemo<I18n>(() => ({ lang, setLang, t: (key, vars) => translate(lang, key, vars) }), [lang, setLang]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useI18n(): I18n {
  const v = useContext(Ctx);
  if (!v) throw new Error('useI18n outside I18nProvider');
  return v;
}
