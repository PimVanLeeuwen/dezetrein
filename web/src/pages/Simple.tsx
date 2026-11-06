import { useI18n } from '../i18n/i18n';
import { Page } from '../components/Page';
import { Link, paths } from '../lib/router';

// Placeholder pages: the trip view comes in step 3, privacy and about in step 5.
export function Train({ train }: { train: string }) {
  const { t } = useI18n();
  return <Page title={t('train.title', { train })} lead={<p>{t('train.soon')}</p>} />;
}

export function Privacy() {
  const { t } = useI18n();
  return <Page title={t('privacy.title')} lead={<p>{t('privacy.soon')}</p>} />;
}

export function About() {
  const { t } = useI18n();
  return <Page title={t('about.title')} lead={<p>{t('about.soon')}</p>} />;
}

export function NotFound() {
  const { t } = useI18n();
  return (
    <Page title={t('notFound.title')} lead={<p>{t('notFound.body')}</p>}>
      <Link to={paths.home} className="mt-6 inline-flex min-h-11 items-center rounded-full bg-brand px-5 font-semibold text-white dark:bg-brand-light dark:text-neutral-950">
        {t('notFound.back')}
      </Link>
    </Page>
  );
}
