import { focusRing } from '../components/Layout';
import { Page } from '../components/Page';
import { useI18n } from '../i18n/i18n';
import { Link, paths } from '../lib/router';
import { ContentPage } from '../components/Prose';
import { about, privacy } from '../content/pages';

export function Privacy() {
  const { lang } = useI18n();
  return <ContentPage content={privacy[lang]} />;
}

export function About() {
  const { lang } = useI18n();
  return <ContentPage content={about[lang]} />;
}

// The not-found page: the rail line runs into a buffer stop.
export function NotFound() {
  const { t } = useI18n();
  return (
    <Page title={t('notFound.title')} lead={<p>{t('notFound.body')}</p>}>
      <div aria-hidden="true" className="rise mt-12 flex items-center" style={{ animationDelay: '160ms' }}>
        <span className="-ml-12 h-[9px] flex-1 rounded-l-full bg-line" />
        <span className="ml-1 h-12 w-3 rounded-sm bg-late" />
        <span className="ml-1 h-8 w-2 rounded-sm bg-late/60" />
      </div>
      <p className="mt-3 text-sm font-bold text-dim">{t('notFound.buffer')}</p>
      <Link to={paths.home} className={`mt-10 inline-flex min-h-12 items-center gap-3 rounded-full bg-line px-6 font-display text-lg font-bold text-on-line ${focusRing}`}>
        {t('notFound.back')} <span aria-hidden="true">→</span>
      </Link>
    </Page>
  );
}
