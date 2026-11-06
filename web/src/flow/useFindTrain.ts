// The "Find my train" flow as a small state machine. It collects location readings while it asks
// the backend: early (after a few seconds) for a quick answer when the match is clear, and again
// at the end of the collection window otherwise. Readings live only in this hook's memory and are
// dropped as soon as the flow ends.
import { useCallback, useEffect, useRef, useState } from 'react';
import { api as defaultClient, ApiError, errorKey, type Client } from '../api/client';
import type { Fix, MatchResult } from '../api/types';
import type { MessageKey } from '../i18n/messages';
import { browserGeo, inArea, maxAccuracyM, spread, type Geo, type GeoError } from '../lib/geo';

export type FailReason = GeoError | 'timeout' | 'outside' | 'none' | 'api';

export type FindState =
  | { phase: 'intro' }
  | { phase: 'locating'; readings: number; accuracyM: number | null; checking: boolean }
  | { phase: 'confirm'; result: MatchResult }
  | { phase: 'failed'; reason: FailReason; message?: MessageKey };

export const timing = {
  firstTryMs: 6000, // ask early: a clear match can be shown right away
  finalMs: 15000, // the full collection window
  giveUpMs: 20000, // no reading at all by now: the phone cannot locate
};

const maxReadings = 30;
const fixesPerRequest = 5;

type Options = { geo?: Geo; client?: Pick<Client, 'match'> };

export function useFindTrain({ geo = browserGeo, client = defaultClient }: Options = {}) {
  const [state, setState] = useState<FindState>({ phase: 'intro' });
  const fixes = useRef<Fix[]>([]);
  const run = useRef(0); // increases per attempt, so late callbacks of an old attempt are ignored
  const stopWatch = useRef<() => void>(() => {});
  const timers = useRef<ReturnType<typeof setTimeout>[]>([]);
  const abort = useRef<AbortController | null>(null);
  const started = useRef(0);
  const finalAsked = useRef(false);

  const halt = useCallback(() => {
    run.current++;
    stopWatch.current();
    stopWatch.current = () => {};
    timers.current.forEach(clearTimeout);
    timers.current = [];
    abort.current?.abort();
    abort.current = null;
    fixes.current = [];
  }, []);

  const finish = useCallback(
    (next: FindState) => {
      halt();
      setState(next);
    },
    [halt],
  );

  const ask = useCallback(
    async (final: boolean, attempt: number) => {
      if (fixes.current.length === 0 || attempt !== run.current) return;
      if (final) finalAsked.current = true;
      abort.current?.abort();
      const ctl = new AbortController();
      abort.current = ctl;
      setState((s) => (s.phase === 'locating' ? { ...s, checking: true } : s));
      try {
        const result = await client.match(spread(fixes.current, fixesPerRequest), ctl.signal);
        if (attempt !== run.current || ctl.signal.aborted) return;
        if (result.candidates.length > 0 && (result.confidence === 'high' || final)) finish({ phase: 'confirm', result });
        else if (final) finish({ phase: 'failed', reason: 'none' });
        else setState((s) => (s.phase === 'locating' ? { ...s, checking: false } : s));
      } catch (err) {
        if (attempt !== run.current || ctl.signal.aborted) return;
        // A failed early try is retried at the end of the window; a failed final one is shown.
        if (final || (err instanceof ApiError && err.status === 429)) finish({ phase: 'failed', reason: 'api', message: errorKey(err) });
        else setState((s) => (s.phase === 'locating' ? { ...s, checking: false } : s));
      }
    },
    [client, finish],
  );

  const start = useCallback(() => {
    halt();
    const attempt = run.current;
    started.current = Date.now();
    finalAsked.current = false;
    setState({ phase: 'locating', readings: 0, accuracyM: null, checking: false });

    stopWatch.current = geo.watch(
      (fix) => {
        if (attempt !== run.current) return;
        if (!inArea(fix)) return finish({ phase: 'failed', reason: 'outside' });
        if (fix.accuracy > maxAccuracyM) return;
        fixes.current = [...fixes.current, fix].slice(-maxReadings);
        const accuracyM = Math.min(...fixes.current.map((f) => f.accuracy));
        setState((s) => (s.phase === 'locating' ? { ...s, readings: fixes.current.length, accuracyM } : s));
        // The window has passed but the first reading came late: ask now.
        if (!finalAsked.current && Date.now() - started.current >= timing.finalMs) void ask(true, attempt);
      },
      (err) => {
        if (attempt !== run.current) return;
        // A position that is briefly unavailable is normal on a train; give up only at the end.
        if (err !== 'unavailable') finish({ phase: 'failed', reason: err });
      },
    );
    timers.current = [
      setTimeout(() => void ask(false, attempt), timing.firstTryMs),
      setTimeout(() => void ask(true, attempt), timing.finalMs),
      setTimeout(() => {
        if (attempt === run.current && fixes.current.length === 0) finish({ phase: 'failed', reason: 'timeout' });
      }, timing.giveUpMs),
    ];
  }, [ask, finish, geo, halt]);

  const reset = useCallback(() => finish({ phase: 'intro' }), [finish]);

  // Skip the explanation when the user already allowed location for this site.
  useEffect(() => {
    let live = true;
    void geo.permission().then((p) => {
      if (live && p === 'granted') start();
    });
    return () => {
      live = false;
      halt();
    };
  }, [geo, halt, start]);

  return { state, start, reset };
}
