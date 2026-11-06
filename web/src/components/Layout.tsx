import type { ReactNode } from 'react';
import { useI18n } from '../i18n/i18n';
import { Link, paths } from '../lib/router';
import { nextTheme, useTheme, type Theme } from '../lib/theme';

export function Logo() {
  return (
    <span className="flex items-center gap-2.5 font-display text-2xl font-extrabold tracking-[-0.03em]">
      <span aria-hidden="true" className="size-3 rounded-full bg-line ring-4 ring-line-soft" />
      dezetrein
    </span>
  );
}

const themeKey = { system: 'theme.system', light: 'theme.light', dark: 'theme.dark' } as const;

function ThemeIcon({ theme }: { theme: Theme }) {
  const common = { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round', 'aria-hidden': true } as const;
  if (theme === 'light')
    return (
      <svg {...common}>
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
      </svg>
    );
  if (theme === 'dark')
    return (
      <svg {...common}>
        <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
      </svg>
    );
  return (
    <svg {...common}>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 3a9 9 0 0 1 0 18z" fill="currentColor" stroke="none" />
    </svg>
  );
}

export const focusRing = 'focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-line';

const chip = `inline-flex min-h-11 min-w-11 items-center justify-center rounded-full border border-rule bg-card px-3.5 text-sm font-bold text-ink transition-colors hover:border-line ${focusRing}`;

export function Layout({ children }: { children: ReactNode }) {
  const { t, lang, setLang } = useI18n();
  const [theme, setTheme] = useTheme();
  const next = nextTheme(theme);
  return (
    <div className="flex min-h-dvh flex-col overflow-x-clip">
      <header className="relative z-10 mx-auto flex w-full max-w-2xl items-center justify-between gap-2 px-5 pt-4 pb-2 sm:px-6">
        <Link to={paths.home} className={`-mx-1 rounded-lg px-1 py-2 ${focusRing}`} aria-label={t('nav.home')}>
          <Logo />
        </Link>
        <div className="flex items-center gap-2">
          <button type="button" className={chip} onClick={() => setLang(lang === 'nl' ? 'en' : 'nl')} aria-label={t('lang.switchLabel')} lang={lang === 'nl' ? 'en' : 'nl'}>
            {lang === 'nl' ? 'EN' : 'NL'}
          </button>
          <button
            type="button"
            className={chip}
            onClick={() => setTheme(next)}
            aria-label={t('theme.next', { current: t(themeKey[theme]), next: t(themeKey[next]) })}
            title={t('theme.current', { name: t(themeKey[theme]) })}
          >
            <ThemeIcon theme={theme} />
          </button>
        </div>
      </header>
      <main id="main" className="relative mx-auto w-full max-w-2xl flex-1 px-5 pb-16 sm:px-6">
        {children}
      </main>
      <footer className="relative mx-auto w-full max-w-2xl border-t border-rule px-5 py-6 text-sm text-dim sm:px-6">
        <nav className="mb-3 flex flex-wrap gap-x-5 gap-y-2 font-bold text-ink" aria-label="footer">
          <Link to={paths.privacy} className={`rounded underline-offset-4 hover:underline ${focusRing}`}>
            {t('nav.privacy')}
          </Link>
          <Link to={paths.about} className={`rounded underline-offset-4 hover:underline ${focusRing}`}>
            {t('nav.about')}
          </Link>
        </nav>
        <p>{t('footer.data')}</p>
      </footer>
    </div>
  );
}
