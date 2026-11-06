import { useI18n } from '../i18n/i18n';
import { Page } from '../components/Page';

// The start screen with the two ways in. The flows behind the buttons follow in the next steps.
export function Home() {
  const { t } = useI18n();
  const action =
    'flex w-full flex-col items-start gap-1 rounded-2xl px-6 py-5 text-left disabled:cursor-not-allowed disabled:opacity-60';
  return (
    <Page title={t('app.tagline')} lead={<p>{t('app.description')}</p>}>
      <div className="mt-8 flex flex-col gap-4">
        <button type="button" disabled className={`${action} bg-brand text-white shadow-sm dark:bg-brand-light dark:text-neutral-950`}>
          <span className="text-xl font-semibold">{t('home.find')}</span>
          <span className="text-base opacity-90">{t('home.findHint')}</span>
        </button>
        <button type="button" disabled className={`${action} border-2 border-neutral-300 bg-white dark:border-neutral-700 dark:bg-neutral-900`}>
          <span className="text-xl font-semibold">{t('home.carriage')}</span>
          <span className="text-base text-neutral-600 dark:text-neutral-400">{t('home.carriageHint')}</span>
        </button>
        <p className="text-sm font-medium uppercase tracking-wide text-brand dark:text-brand-light">{t('home.soon')}</p>
      </div>
    </Page>
  );
}
