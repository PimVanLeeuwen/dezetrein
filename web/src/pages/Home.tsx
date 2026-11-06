import { useRef, useState, type ReactNode, type RefObject } from 'react';
import { focusRing } from '../components/Layout';
import { Page } from '../components/Page';
import { RailLine } from '../components/RailLine';
import { useI18n } from '../i18n/i18n';

// The start screen: two ways in, both stations on the rail line. The flows behind them follow in
// the next steps; until then the buttons say so and do nothing.
function Stop({ stopRef, primary, title, hint, mark, delay }: { stopRef: RefObject<HTMLButtonElement | null>; primary?: boolean; title: string; hint: string; mark: ReactNode; delay: number }) {
  const look = primary
    ? 'border-line bg-line text-on-line shadow-[0_22px_44px_-20px_var(--line)]'
    : 'border-rule bg-card text-ink shadow-[0_14px_30px_-22px_rgba(20,32,27,0.45)] hover:border-line';
  return (
    <button
      ref={stopRef}
      type="button"
      aria-disabled="true"
      className={`rise flex w-full items-center gap-4 rounded-3xl border-[1.5px] px-5 py-5 text-left transition-transform active:scale-[0.98] ${look} ${focusRing}`}
      style={{ animationDelay: `${delay}ms` }}
    >
      <span className="flex-1">
        <span className="block font-display text-[1.4rem] leading-tight font-bold tracking-[-0.02em]">{title}</span>
        <span className="mt-0.5 block text-base font-medium opacity-80">{hint}</span>
      </span>
      <span aria-hidden="true" className="grid size-11 shrink-0 place-items-center rounded-full bg-current/12 text-xl font-bold">
        {mark}
      </span>
    </button>
  );
}

export function Home() {
  const { t } = useI18n();
  const area = useRef<HTMLDivElement>(null);
  const find = useRef<HTMLButtonElement>(null);
  const carriage = useRef<HTMLButtonElement>(null);
  const [stops] = useState(() => [find, carriage]);
  return (
    <Page
      title={t('app.tagline')}
      heading={
        <>
          {t('home.titleStart')} <span className="text-line">{t('home.titleEnd')}</span>
        </>
      }
      lead={<p>{t('home.lead')}</p>}
    >
      <div ref={area} className="relative mt-24 pr-14 pb-24 sm:mt-28 sm:pr-20">
        <RailLine container={area} stops={stops} />
        <div className="relative flex flex-col gap-4">
          <Stop stopRef={find} primary title={t('home.find')} hint={t('home.findHint')} mark="→" delay={200} />
          <Stop stopRef={carriage} title={t('home.carriage')} hint={t('home.carriageHint')} mark="#" delay={280} />
          <p className="rise mt-1 text-sm font-bold tracking-wide text-line uppercase" style={{ animationDelay: '360ms' }}>
            {t('home.soon')}
          </p>
        </div>
      </div>
      <p className="text-sm font-medium text-dim">{t('home.privacyNote')}</p>
    </Page>
  );
}
