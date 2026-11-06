import type { ReactNode } from 'react';
import type { Candidate } from '../api/types';
import { focusRing } from '../components/Layout';
import { Page } from '../components/Page';
import { primary, secondary, textLink } from '../components/ui';
import { useFindTrain, type FailReason, type FindState } from '../flow/useFindTrain';
import { useI18n } from '../i18n/i18n';
import { nl, type MessageKey } from '../i18n/messages';
import type { Geo } from '../lib/geo';
import { Link, navigate, paths } from '../lib/router';
import type { Client } from '../api/client';


// useCategory turns a train category code (IC, SPR, ...) into its name, or the code itself when
// the dictionary has none.
function useCategory() {
  const { t } = useI18n();
  return (code?: string) => {
    if (!code) return '';
    const key = `category.${code}`;
    return key in nl ? t(key as MessageKey) : code;
  };
}

// The short track the train rides along while the phone locates and the backend compares.
function Track({ label }: { label: string }) {
  return (
    <div className="mt-12" role="status" aria-live="polite">
      <div aria-hidden="true" className="relative h-7">
        <span className="absolute inset-x-0 top-1/2 h-[9px] -translate-y-1/2 rounded-full bg-line-soft" />
        <span className="track-ride absolute top-0 size-7 rounded-full border-[5px] border-paper bg-sun shadow-[0_0_0_6px_color-mix(in_srgb,var(--sun)_30%,transparent)]" />
      </div>
      <p className="mt-5 text-lg font-bold">{label}</p>
    </div>
  );
}

function Carriages({ list }: { list: string[] }) {
  const { t } = useI18n();
  if (list.length === 0) return <p className="mt-3 text-sm font-medium text-dim">{t('find.noCarriages')}</p>;
  return (
    <p className="mt-4 flex flex-wrap items-center gap-2" aria-label={`${t('find.carriages')}: ${list.join(', ')}`}>
      {list.map((c) => (
        <span key={c} className="rounded-lg border-2 border-ink px-2.5 py-1 font-mono text-base font-bold">
          {c}
        </span>
      ))}
    </p>
  );
}

function CandidateCard({ c, onPick }: { c: Candidate; onPick: (train: string) => void }) {
  const { t } = useI18n();
  const category = useCategory();
  return (
    <button
      type="button"
      onClick={() => onPick(c.train)}
      className={`w-full rounded-3xl border-[1.5px] border-rule bg-card px-5 py-4 text-left transition-colors hover:border-line ${focusRing}`}
    >
      <span className="block text-sm font-bold text-dim">{t('find.train', { category: category(c.category), train: c.train })}</span>
      <span className="mt-1 block font-display text-2xl font-bold tracking-[-0.02em]">{c.destination ?? t('find.unknownDestination')}</span>
      <Carriages list={c.carriages} />
    </button>
  );
}

const failure: Record<FailReason, [MessageKey, MessageKey | null]> = {
  denied: ['find.deniedTitle', 'find.deniedBody'],
  unsupported: ['find.unsupportedTitle', 'find.unsupportedBody'],
  unavailable: ['find.timeoutTitle', 'find.timeoutBody'],
  timeout: ['find.timeoutTitle', 'find.timeoutBody'],
  outside: ['find.outsideTitle', 'find.outsideBody'],
  none: ['find.noneTitle', 'find.noneBody'],
  api: ['find.apiTitle', null],
};

function Actions({ children }: { children: ReactNode }) {
  return <div className="mt-10 flex flex-col gap-3 sm:flex-row">{children}</div>;
}

function formatAccuracy(m: number, lang: string) {
  if (m >= 1000) return `${(m / 1000).toLocaleString(lang, { maximumFractionDigits: 1 })} km`;
  return `${Math.round(m)} m`;
}

export function FindView({ state, start, reset }: { state: FindState; start: () => void; reset: () => void }) {
  const { t, lang } = useI18n();
  const category = useCategory();
  const pick = (train: string) => navigate(paths.train(train));

  if (state.phase === 'intro') {
    const points: MessageKey[] = ['find.pointRough', 'find.pointPrivate', 'find.pointConfirm'];
    return (
      <Page title={t('find.title')} lead={<p>{t('find.lead')}</p>}>
        <ul className="mt-8 flex flex-col gap-3">
          {points.map((k, i) => (
            <li key={k} className="rise flex gap-3 text-base font-medium" style={{ animationDelay: `${120 + i * 60}ms` }}>
              <span aria-hidden="true" className="mt-1.5 size-3 shrink-0 rounded-full border-[3px] border-line" />
              {t(k)}
            </li>
          ))}
        </ul>
        <Actions>
          <button type="button" className={primary} onClick={start}>
            {t('find.share')} <span aria-hidden="true">→</span>
          </button>
          <Link to={paths.home} className={secondary}>
            {t('find.back')}
          </Link>
        </Actions>
      </Page>
    );
  }

  if (state.phase === 'locating') {
    const label = state.checking ? t('find.checking') : state.readings === 0 ? t('find.waiting') : t('find.readings', { count: state.readings, accuracy: formatAccuracy(state.accuracyM ?? 0, lang) });
    return (
      <Page title={t('find.locatingTitle')} lead={<p>{t('find.lead')}</p>}>
        <Track label={label} />
        {state.accuracyM !== null && state.accuracyM > 500 && <p className="mt-2 text-base font-medium text-dim">{t('find.rough')}</p>}
        <Actions>
          <button type="button" className={secondary} onClick={reset}>
            {t('find.cancel')}
          </button>
        </Actions>
      </Page>
    );
  }

  if (state.phase === 'confirm') {
    const [top, ...others] = state.result.candidates;
    if (!top) return null;
    const question = top.destination ? t('find.question', { destination: top.destination }) : t('find.questionNumber', { train: top.train });
    return (
      <Page title={question} compact>
        <div className="rise mt-8 rounded-3xl border-[1.5px] border-line bg-card p-5 shadow-[0_18px_40px_-26px_var(--line)]" style={{ animationDelay: '80ms' }}>
          <p className="text-base font-bold text-dim">{t('find.train', { category: category(top.category), train: top.train })}</p>
          <Carriages list={top.carriages} />
        </div>
        <Actions>
          <button type="button" className={primary} onClick={() => pick(top.train)}>
            {t('find.yes')} <span aria-hidden="true">→</span>
          </button>
          <button type="button" className={secondary} onClick={start}>
            {t('find.no')}
          </button>
        </Actions>
        {others.length > 0 && (
          <section className="mt-12" aria-labelledby="others">
            <h2 id="others" className="font-display text-2xl font-bold tracking-[-0.02em]">
              {t('find.others')}
            </h2>
            <div className="mt-4 flex flex-col gap-3">
              {others.map((c) => (
                <CandidateCard key={c.train} c={c} onPick={pick} />
              ))}
            </div>
          </section>
        )}
      </Page>
    );
  }

  const [title, body] = failure[state.reason];
  return (
    <Page title={t(title)} lead={<p>{body ? t(body) : t(state.message ?? 'error.unknown')}</p>}>
      <Actions>
        {state.reason !== 'unsupported' && state.reason !== 'outside' && (
          <button type="button" className={primary} onClick={start}>
            {t('find.retry')}
          </button>
        )}
        <Link to={paths.carriage} className={secondary}>
          {t('find.byCarriage')}
        </Link>
      </Actions>
      <p className="mt-6">
        <Link to={paths.other} className={textLink}>
          {t('find.otherWays')}
        </Link>
      </p>
    </Page>
  );
}

export function Find({ geo, client }: { geo?: Geo; client?: Pick<Client, 'match'> }) {
  const flow = useFindTrain({ geo, client });
  return <FindView {...flow} />;
}
