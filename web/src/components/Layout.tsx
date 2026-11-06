import type { ReactNode } from 'react';
import { useI18n } from '../i18n/i18n';
import { Link, paths } from '../lib/router';
import { nextTheme, useTheme, type Theme } from '../lib/theme';

export function Logo() {
  return (
    <span className="text-xl font-bold tracking-tight">
      deze<span className="text-brand dark:text-brand-light">trein</span>
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
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M8 20h8M12 16v4" />
    </svg>
  );
}

const control =
  'inline-flex min-h-11 min-w-11 items-center justify-center gap-2 rounded-full px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-200/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand dark:text-neutral-200 dark:hover:bg-neutral-800 dark:focus-visible:outline-brand-light';

export function Layout({ children }: { children: ReactNode }) {
  const { t, lang, setLang } = useI18n();
  const [theme, setTheme] = useTheme();
  const next = nextTheme(theme);
  return (
    <div className="flex min-h-dvh flex-col bg-neutral-50 font-sans text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="mx-auto flex w-full max-w-2xl items-center justify-between gap-2 px-4 py-3">
        <Link to={paths.home} className="rounded-md px-1 py-2 focus-visible:outline-2 focus-visible:outline-brand dark:focus-visible:outline-brand-light" aria-label={t('nav.home')}>
          <Logo />
        </Link>
        <div className="flex items-center gap-1">
          <button type="button" className={control} onClick={() => setLang(lang === 'nl' ? 'en' : 'nl')} aria-label={t('lang.switchLabel')} lang={lang === 'nl' ? 'en' : 'nl'}>
            {t('lang.switch')}
          </button>
          <button
            type="button"
            className={control}
            onClick={() => setTheme(next)}
            aria-label={t('theme.next', { current: t(themeKey[theme]), next: t(themeKey[next]) })}
            title={t('theme.current', { name: t(themeKey[theme]) })}
          >
            <ThemeIcon theme={theme} />
          </button>
        </div>
      </header>
      <main id="main" className="mx-auto w-full max-w-2xl flex-1 px-4 pb-12">
        {children}
      </main>
      <footer className="mx-auto w-full max-w-2xl px-4 py-6 text-sm text-neutral-600 dark:text-neutral-400">
        <nav className="mb-2 flex gap-4" aria-label="footer">
          <Link to={paths.privacy} className="underline-offset-4 hover:underline">
            {t('nav.privacy')}
          </Link>
          <Link to={paths.about} className="underline-offset-4 hover:underline">
            {t('nav.about')}
          </Link>
        </nav>
        <p>{t('footer.data')}</p>
      </footer>
    </div>
  );
}
