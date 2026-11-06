import { useCallback, useEffect, useId, useRef, useState, type FormEvent } from 'react';
import { api as defaultClient, errorKey, type Client } from '../api/client';
import type { Departure, NearbyStation } from '../api/types';
import { Page } from '../components/Page';
import { card, formatDistance, numberInput, onlyDigits, primary, secondary } from '../components/ui';
import { useI18n } from '../i18n/i18n';
import type { MessageKey } from '../i18n/messages';
import { browserGeo, sample, type Geo } from '../lib/geo';
import { navigate, paths } from '../lib/router';
import { hhmm } from '../lib/trip';

// Other ways in: a journey number typed directly, or the departures at the nearest station.

type Departures =
  | { phase: 'idle' }
  | { phase: 'locating' }
  | { phase: 'stations'; stations: NearbyStation[] }
  | { phase: 'loading'; station: NearbyStation; stations: NearbyStation[] }
  | { phase: 'departures'; station: NearbyStation; departures: Departure[]; stations: NearbyStation[] }
  | { phase: 'failed'; message: MessageKey };

const geoMessage: Record<string, MessageKey> = {
  denied: 'find.deniedBody',
  unsupported: 'find.unsupportedBody',
  timeout: 'find.timeoutBody',
  outside: 'find.outsideBody',
};

export function useStationDepartures(client: Pick<Client, 'nearestStations' | 'departures'> = defaultClient, geo: Geo = browserGeo) {
  const [state, setState] = useState<Departures>({ phase: 'idle' });
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const fresh = () => {
    abort.current?.abort();
    abort.current = new AbortController();
    return abort.current.signal;
  };

  const findStations = useCallback(async () => {
    const signal = fresh();
    setState({ phase: 'locating' });
    let fixes;
    try {
      fixes = await sample(geo, { want: 1, maxMs: 8000, signal });
    } catch (err) {
      if (!signal.aborted) setState({ phase: 'failed', message: geoMessage[(err as Error).message] ?? 'error.unknown' });
      return;
    }
    try {
      const res = await client.nearestStations(fixes, signal);
      if (!signal.aborted) setState({ phase: 'stations', stations: res.stations });
    } catch (err) {
      if (!signal.aborted) setState({ phase: 'failed', message: errorKey(err) });
    }
  }, [client, geo]);

  const pick = useCallback(
    async (station: NearbyStation, stations: NearbyStation[]) => {
      const signal = fresh();
      setState({ phase: 'loading', station, stations });
      try {
        const res = await client.departures(station.code, signal);
        if (!signal.aborted) setState({ phase: 'departures', station, departures: res.departures, stations });
      } catch (err) {
        if (signal.aborted) return;
        // Back to the list with the error, so another station can be tried.
        setState(stations.length ? { phase: 'stations', stations } : { phase: 'failed', message: errorKey(err) });
      }
    },
    [client],
  );

  const back = useCallback((stations: NearbyStation[]) => setState({ phase: 'stations', stations }), []);
  return { state, findStations, pick, back };
}

function JourneyNumber() {
  const { t } = useI18n();
  const [value, setValue] = useState('');
  const [invalid, setInvalid] = useState(false);
  const id = useId();
  const submit = (e: FormEvent) => {
    e.preventDefault();
    const n = onlyDigits(value);
    if (n.length < 1 || n.length > 6) return setInvalid(true);
    navigate(paths.train(n));
  };
  return (
    <section className="mt-10">
      <h2 className="font-display text-2xl font-bold tracking-[-0.02em]">
        {t('other.journeyTitle')}
      </h2>
      <p className="mt-1 text-base text-dim">{t('other.journeyHint')}</p>
      <form className="mt-4 flex flex-col gap-3 sm:flex-row" onSubmit={submit} noValidate>
        <label htmlFor={id} className="sr-only">
          {t('other.journeyLabel')}
        </label>
        <input
          id={id}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          inputMode="numeric"
          autoComplete="off"
          maxLength={8}
          placeholder="3522"
          aria-invalid={invalid}
          aria-describedby={invalid ? `${id}-error` : undefined}
          className={`${numberInput} sm:max-w-56`}
        />
        <button type="submit" className={primary}>
          {t('other.journeySubmit')} <span aria-hidden="true">→</span>
        </button>
      </form>
      {invalid && (
        <p id={`${id}-error`} className="mt-2 font-bold text-late">
          {t('carriage.invalid')}
        </p>
      )}
    </section>
  );
}

function DepartureRow({ d }: { d: Departure }) {
  const { t } = useI18n();
  const late = d.actual ? Math.round((Date.parse(d.actual) - Date.parse(d.planned)) / 60000) : 0;
  const track = d.actualTrack || d.plannedTrack || '';
  return (
    <li>
      <button
        type="button"
        onClick={() => navigate(paths.train(d.train))}
        className={`${card} flex items-center gap-4`}
        aria-label={t('other.departureRow', { time: hhmm(d.actual ?? d.planned), category: d.category, direction: d.direction, track: track ? t('other.track', { track }) : '' })}
      >
        <span className="w-16 shrink-0">
          <span className={`block font-mono text-xl font-bold ${late > 0 ? 'text-late' : ''} ${d.cancelled ? 'line-through' : ''}`}>{hhmm(d.actual ?? d.planned)}</span>
          {late > 0 && <span className="block font-mono text-sm text-dim line-through">{hhmm(d.planned)}</span>}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-lg font-bold">{d.direction}</span>
          <span className="block text-sm font-bold text-dim">
            {d.category} {d.train}
          </span>
        </span>
        {track && (
          <span className={`grid h-9 min-w-9 place-items-center rounded-xl px-1.5 font-display text-lg font-extrabold ${d.actualTrack && d.plannedTrack && d.actualTrack !== d.plannedTrack ? 'bg-late text-on-late' : 'bg-ink text-paper'}`}>
            {track}
          </span>
        )}
      </button>
    </li>
  );
}

function StationDepartures({ flow }: { flow: ReturnType<typeof useStationDepartures> }) {
  const { t, lang } = useI18n();
  const { state } = flow;
  const status = (text: string) => (
    <p className="mt-4 text-lg font-bold" role="status" aria-live="polite">
      {text}
    </p>
  );
  return (
    <section className="mt-12" aria-labelledby="departures-h">
      <h2 id="departures-h" className="font-display text-2xl font-bold tracking-[-0.02em]">
        {state.phase === 'departures' || state.phase === 'loading' ? t('other.departuresAt', { station: state.station.name }) : t('other.departuresTitle')}
      </h2>
      {(state.phase === 'idle' || state.phase === 'failed') && (
        <>
          <p className="mt-1 text-base text-dim">{t('other.departuresHint')}</p>
          {state.phase === 'failed' && <p className="mt-3 font-bold text-late">{t(state.message)}</p>}
          <button type="button" className={`mt-4 ${secondary}`} onClick={() => void flow.findStations()}>
            {t('other.findStations')}
          </button>
        </>
      )}
      {state.phase === 'locating' && status(t('other.locating'))}
      {state.phase === 'loading' && status(t('other.loadingDepartures'))}
      {state.phase === 'stations' &&
        (state.stations.length === 0 ? (
          status(t('other.noStations'))
        ) : (
          <ul className="mt-4 flex flex-col gap-3">
            {state.stations.map((s) => (
              <li key={s.code}>
                <button type="button" className={card} onClick={() => void flow.pick(s, state.stations)}>
                  <span className="block text-lg font-bold">{s.name}</span>
                  <span className="block text-sm font-bold text-dim">{t('other.distance', { distance: formatDistance(s.distanceM, lang) })}</span>
                </button>
              </li>
            ))}
          </ul>
        ))}
      {state.phase === 'departures' && (
        <>
          {state.departures.length === 0 ? status(t('other.noDepartures')) : (
            <ul className="mt-4 flex flex-col gap-2">
              {state.departures.slice(0, 12).map((d) => (
                <DepartureRow key={`${d.train}-${d.planned}`} d={d} />
              ))}
            </ul>
          )}
          {state.stations.length > 1 && (
            <button type="button" className={`mt-4 ${secondary}`} onClick={() => flow.back(state.stations)}>
              {t('other.otherStation')}
            </button>
          )}
        </>
      )}
    </section>
  );
}

export function Other({ client, geo }: { client?: Pick<Client, 'nearestStations' | 'departures'>; geo?: Geo }) {
  const { t } = useI18n();
  const flow = useStationDepartures(client, geo);
  return (
    <Page title={t('other.title')} lead={<p>{t('other.lead')}</p>}>
      <JourneyNumber />
      <StationDepartures flow={flow} />
    </Page>
  );
}
