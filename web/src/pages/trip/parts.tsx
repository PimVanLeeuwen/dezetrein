// The building blocks of the trip view. Station names and NS texts are shown as NS publishes
// them (Dutch); everything else comes from the dictionary.
import type { ReactNode } from 'react';
import type { Composition, Disruption, Stop } from '../../api/types';
import { focusRing } from '../../components/Layout';
import { useI18n } from '../../i18n/i18n';
import type { MessageKey } from '../../i18n/messages';
import { cancelled, crowdLevel, delayMinutes, hhmm, mainEvent, platform } from '../../lib/trip';

export function Pill({ tone = 'plain', children, label }: { tone?: 'plain' | 'late' | 'good'; children: ReactNode; label?: string }) {
  const look = tone === 'late' ? 'border-late bg-late text-on-late' : tone === 'good' ? 'border-line-soft bg-line-soft text-ink' : 'border-rule bg-card text-ink';
  return (
    <span aria-label={label} className={`inline-flex min-h-9 items-center rounded-full border px-3.5 text-sm font-bold ${look}`}>
      {children}
    </span>
  );
}

// Crowd shows NS's forecast as three bars, with the level in words for screen readers.
export function Crowd({ crowd }: { crowd?: string }) {
  const { t } = useI18n();
  const level = crowdLevel(crowd);
  if (level === 0) return null;
  const name = t(`trip.crowd${level}` as MessageKey);
  return (
    <span className="inline-flex items-center gap-1.5 text-sm font-bold text-dim" title={t('trip.crowd', { level: name })}>
      <span aria-hidden="true" className="flex items-end gap-[3px]">
        {[1, 2, 3].map((b) => (
          <span key={b} className={`w-[5px] rounded-sm ${b <= level ? (level === 3 ? 'bg-late' : 'bg-line') : 'bg-rule'}`} style={{ height: `${6 + b * 4}px` }} />
        ))}
      </span>
      <span>{name}</span>
    </span>
  );
}

function Platform({ stop, big }: { stop: Stop; big?: boolean }) {
  const { t } = useI18n();
  const p = platform(mainEvent(stop));
  if (!p.track) return null;
  return (
    <span className="flex items-center gap-2 text-sm font-bold">
      <span className="text-dim">{t('trip.platform')}</span>
      <span
        className={`grid place-items-center rounded-xl font-display font-extrabold ${big ? 'h-12 min-w-12 px-2 text-2xl' : 'h-8 min-w-8 px-1.5 text-base'} ${p.changed ? 'bg-late text-on-late' : 'bg-ink text-paper'}`}
      >
        {p.track}
      </span>
    </span>
  );
}

// Time shows the expected time, with the planned time struck through when it differs.
function Time({ stop, big }: { stop: Stop; big?: boolean }) {
  const { t } = useI18n();
  const e = mainEvent(stop);
  if (!e) return null;
  const late = delayMinutes(e) > 0;
  const shown = hhmm(e.actual ?? e.planned);
  return (
    <span className="flex items-baseline gap-2">
      <span className={`font-mono font-bold tabular-nums ${big ? 'text-4xl' : 'text-lg'} ${late ? 'text-late' : ''}`}>{shown}</span>
      {late && (
        <span className={`font-mono text-dim line-through ${big ? 'text-lg' : 'text-sm'}`} aria-label={t('trip.planned', { time: hhmm(e.planned) })}>
          {hhmm(e.planned)}
        </span>
      )}
    </span>
  );
}

export function NextStop({ stop, label }: { stop: Stop; label: string }) {
  const { t } = useI18n();
  const p = platform(mainEvent(stop));
  const off = cancelled(stop);
  return (
    <div className="rise relative rounded-3xl border-[1.5px] border-rule bg-card p-5 shadow-[0_18px_40px_-28px_rgba(20,32,27,0.5)]" style={{ animationDelay: '120ms' }}>
      <p className="text-sm font-bold tracking-wide text-dim uppercase">{label}</p>
      <p className={`mt-1 font-display text-3xl leading-tight font-extrabold tracking-[-0.03em] ${off ? 'line-through decoration-late decoration-2' : ''}`}>{stop.name}</p>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <Time stop={stop} big />
        <Platform stop={stop} big />
      </div>
      {p.changed && <p className="mt-3 text-base font-bold text-late">{t('trip.platformChanged', { planned: p.planned ?? '' })}</p>}
      {off && <p className="mt-3 text-base font-bold text-late">{t('trip.cancelled')}</p>}
      {!off && stop.crowd && (
        <div className="mt-3">
          <Crowd crowd={stop.crowd} />
        </div>
      )}
    </div>
  );
}

export function StopRow({ stop, past }: { stop: Stop; past?: boolean }) {
  const { t } = useI18n();
  const off = cancelled(stop);
  const p = platform(mainEvent(stop));
  return (
    <li className={`relative py-3 ${past ? 'text-dim' : ''}`}>
      <span aria-hidden="true" className={`absolute top-[1.35rem] -left-[2.15rem] size-[15px] rounded-full border-4 bg-paper ${past ? 'border-line-soft' : 'border-line'} ${stop.status === 'DESTINATION' ? 'size-[19px] -left-[2.4rem]' : ''}`} />
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className={`text-lg leading-snug font-bold ${off ? 'line-through decoration-late decoration-2' : ''}`}>{stop.name}</p>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1">
            <Time stop={stop} />
            {!past && !off && <Crowd crowd={stop.crowd} />}
          </div>
          {off && <p className="text-sm font-bold text-late">{t('trip.cancelled')}</p>}
          {!past && p.changed && <p className="text-sm font-bold text-late">{t('trip.platformChanged', { planned: p.planned ?? '' })}</p>}
        </div>
        {!past && <Platform stop={stop} />}
      </div>
    </li>
  );
}

// The train drawn as its trainsets: rounded noses at both ends of the whole train.
function carShape(i: number, n: number) {
  if (n === 1) return 'rounded-[1.75rem]';
  if (i === 0) return 'rounded-l-[1.75rem] rounded-r-md';
  if (i === n - 1) return 'rounded-l-md rounded-r-[1.75rem]';
  return 'rounded-md';
}

// mine is the trainset the user searched by: marked, so they can see it is the right train.
export function Carriages({ composition, mine }: { composition: Composition; mine?: string }) {
  const { t } = useI18n();
  const n = composition.carriages.length;
  if (n === 0) return null;
  return (
    <section className="mt-10" aria-labelledby="stock">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="stock" className="font-display text-2xl font-bold tracking-[-0.02em]">
          {t('trip.stock')}
        </h2>
        <p className="text-sm font-bold text-dim">
          {[composition.type, n === 1 ? t('trip.stockOne') : t('trip.stockCount', { count: n }), composition.lengthM ? t('trip.length', { meters: composition.lengthM }) : null].filter(Boolean).join(' · ')}
        </p>
      </div>
      <ol className="mt-4 flex gap-1" aria-label={`${t('trip.stock')}: ${composition.carriages.join(', ')}`}>
        {composition.carriages.map((c, i) => (
          <li
            key={`${c}-${i}`}
            className={`relative grid h-14 min-w-0 flex-1 place-items-center border-2 font-mono text-lg font-bold ${carShape(i, n)} ${c === mine ? 'border-line bg-line text-on-line' : 'border-ink bg-card'}`}
            aria-current={c === mine ? 'true' : undefined}
          >
            {c}
            {c === mine && <span className="absolute -bottom-6 font-sans text-xs font-bold tracking-wide text-line uppercase">{t('trip.yours')}</span>}
          </li>
        ))}
      </ol>
    </section>
  );
}

function DisruptionCard({ d, kind }: { d: Disruption; kind: string }) {
  const { t } = useI18n();
  return (
    <section className="rise rounded-2xl border-[1.5px] border-l-8 border-rule border-l-late bg-card px-4 py-3.5" style={{ animationDelay: '80ms' }} aria-label={`${kind}: ${d.title}`}>
      <p className="text-xs font-bold tracking-widest text-late uppercase">{kind}</p>
      <p className="mt-0.5 text-base font-bold">{d.title}</p>
      {d.description && <p className="mt-1 text-sm text-dim">{d.description}</p>}
      {d.url?.startsWith('https://') && (
        <a href={d.url} target="_blank" rel="noopener noreferrer" className={`mt-2 inline-block rounded text-sm font-bold text-line underline underline-offset-4 ${focusRing}`}>
          {t('trip.moreAtNs')}
        </a>
      )}
    </section>
  );
}

const kinds = ['DISRUPTION', 'MAINTENANCE', 'CALAMITY'];

function useKind() {
  const { t } = useI18n();
  return (d: Disruption) => t(`trip.disruption.${kinds.includes(d.type) ? d.type : 'CALAMITY'}` as MessageKey);
}

// Disruptions on the route come first: they change this trip. Engineering works and NS's general
// notices (which often concern other lines) are listed below the trip, folded away.
export function Disruptions({ list }: { list: Disruption[] }) {
  const kind = useKind();
  const urgent = list.filter((d) => d.type === 'DISRUPTION');
  if (urgent.length === 0) return null;
  return (
    <div className="mt-6 flex flex-col gap-3">
      {urgent.map((d) => (
        <DisruptionCard key={d.id} d={d} kind={kind(d)} />
      ))}
    </div>
  );
}

export function Notices({ list }: { list: Disruption[] }) {
  const { t } = useI18n();
  const kind = useKind();
  const rest = list.filter((d) => d.type !== 'DISRUPTION');
  if (rest.length === 0) return null;
  return (
    <details className="group mt-10 rounded-2xl border-[1.5px] border-rule bg-card">
      <summary className={`flex min-h-12 cursor-pointer list-none items-center justify-between gap-3 rounded-2xl px-4 py-3 font-bold ${focusRing}`}>
        {t('trip.notices', { count: rest.length })}
        <span aria-hidden="true" className="text-xl transition-transform group-open:rotate-180">
          ⌄
        </span>
      </summary>
      <div className="flex flex-col gap-3 px-3 pb-3">
        {rest.map((d) => (
          <DisruptionCard key={d.id} d={d} kind={kind(d)} />
        ))}
      </div>
    </details>
  );
}
