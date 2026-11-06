import { useState } from 'react';
import type { Client } from '../api/client';
import type { Trip as TripData } from '../api/types';
import { focusRing } from '../components/Layout';
import { Page } from '../components/Page';
import { useTrip, type TripState } from '../flow/useTrip';
import { useI18n } from '../i18n/i18n';
import { Link, paths } from '../lib/router';
import { currentDelay, hhmm, progress } from '../lib/trip';
import { Carriages, Disruptions, NextStop, Notices, Pill, StopRow } from './trip/parts';

const button = `inline-flex min-h-14 items-center justify-center gap-3 rounded-full bg-line px-6 font-display text-xl font-bold text-on-line ${focusRing}`;

function Skeleton() {
  const { t } = useI18n();
  return (
    <div className="pt-8 sm:pt-14" role="status" aria-label={t('trip.loading')}>
      <div className="h-4 w-40 animate-pulse rounded bg-rule motion-reduce:animate-none" />
      <div className="mt-4 h-12 w-4/5 animate-pulse rounded-xl bg-rule motion-reduce:animate-none" />
      <div className="mt-8 h-44 animate-pulse rounded-3xl bg-rule motion-reduce:animate-none" />
    </div>
  );
}

function TripBody({ trip, now, refreshFailed }: { trip: TripData; now: number; refreshFailed: boolean }) {
  const { t } = useI18n();
  const [showPast, setShowPast] = useState(false);
  const p = progress(trip.stops, now);
  const delay = currentDelay(p);
  const nextLabel = p.state === 'before' ? t('trip.departure') : p.next?.status === 'DESTINATION' ? t('trip.terminus') : t('trip.next');
  const title = t('trip.titleFull', { train: trip.train, destination: trip.destination });
  return (
    <Page
      compact
      title={title}
      heading={
        <>
          <span className="mb-2 block font-sans text-base font-bold tracking-normal text-dim">
            {trip.categoryName || trip.category} {trip.train}
          </span>
          <span className="text-dim">{t('trip.to')}</span> {trip.destination}
        </>
      }
    >
      <div className="rise mt-5 flex flex-wrap gap-2" style={{ animationDelay: '60ms' }}>
        {p.state === 'arrived' ? (
          <Pill tone="good">{t('trip.arrived')}</Pill>
        ) : delay > 0 ? (
          <Pill tone="late" label={t('trip.lateLabel', { minutes: delay })}>
            {t('trip.late', { minutes: delay })}
          </Pill>
        ) : (
          <Pill tone="good">{t('trip.onTime')}</Pill>
        )}
        {p.state === 'before' && <Pill>{t('trip.before')}</Pill>}
        {trip.composition?.type && <Pill>{trip.composition.type}</Pill>}
      </div>

      <Disruptions list={trip.disruptions} />

      {/* The route as the rail line: stops behind you on the faded part, then you, then ahead. */}
      <div className="relative mt-8 pl-12">
        <span aria-hidden="true" className="absolute top-2 bottom-6 left-[0.9rem] w-[9px] rounded-full bg-line" />
        {p.past.length > 0 && (
          <>
            <span aria-hidden="true" className="absolute top-2 left-[0.9rem] h-12 w-[9px] rounded-full bg-line-soft" />
            <button type="button" onClick={() => setShowPast((v) => !v)} aria-expanded={showPast} className={`mb-2 min-h-11 rounded-lg text-left text-sm font-bold text-dim underline-offset-4 hover:underline ${focusRing}`}>
              {showPast ? t('trip.hidePast') : p.past.length === 1 ? t('trip.pastOne') : t('trip.past', { count: p.past.length })}
            </button>
            {showPast && (
              <ol className="mb-2">
                {p.past.map((s) => (
                  <StopRow key={`${s.code}-${s.uic}`} stop={s} past />
                ))}
              </ol>
            )}
          </>
        )}
        {p.state === 'running' && (
          <p className="relative mb-3 text-xs font-bold tracking-widest text-dim uppercase">
            <span aria-hidden="true" className="absolute top-1/2 -left-[2.55rem] size-[26px] -translate-y-1/2 animate-pulse rounded-full border-4 border-paper bg-sun shadow-[0_0_0_6px_color-mix(in_srgb,var(--sun)_30%,transparent)] motion-reduce:animate-none" />
            {t('trip.youAreHere')}
          </p>
        )}
        {p.next && <NextStop stop={p.next} label={nextLabel} />}
        {p.later.length > 0 && (
          <ol className="mt-3">
            {p.later.map((s) => (
              <StopRow key={`${s.code}-${s.uic}`} stop={s} />
            ))}
          </ol>
        )}
      </div>

      {trip.composition && <Carriages composition={trip.composition} />}

      <Notices list={trip.disruptions} />

      <div className="mt-10 flex flex-col gap-2 text-sm font-medium text-dim" aria-live="polite">
        <p>{t('trip.asOf', { time: hhmm(trip.as_of) })}</p>
        {trip.stale && <p className="font-bold text-late">{t('trip.stale')}</p>}
        {refreshFailed && <p className="font-bold text-late">{t('trip.refreshFailed')}</p>}
      </div>
    </Page>
  );
}

export function TripView({ train, state, now, reload }: { train: string; state: TripState; now: number; reload: () => void }) {
  const { t } = useI18n();
  if (state.status === 'loading') return <Skeleton />;
  if (state.status === 'notFound')
    return (
      <Page compact title={t('trip.notFoundTitle', { train })} lead={<p>{t('trip.notFoundBody')}</p>}>
        <Link to={paths.find} className={`mt-10 ${button}`}>
          {t('trip.find')} <span aria-hidden="true">→</span>
        </Link>
      </Page>
    );
  if (state.status === 'failed')
    return (
      <Page compact title={t('find.apiTitle')} lead={<p>{t(state.message)}</p>}>
        <button type="button" onClick={reload} className={`mt-10 ${button}`}>
          {t('find.retry')}
        </button>
      </Page>
    );
  return <TripBody trip={state.trip} now={now} refreshFailed={state.refreshFailed} />;
}

export function Trip({ train, client }: { train: string; client?: Pick<Client, 'trip'> }) {
  const { state, now, reload } = useTrip(train, client);
  return <TripView train={train} state={state} now={now} reload={() => void reload()} />;
}
