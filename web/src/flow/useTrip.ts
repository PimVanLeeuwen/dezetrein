// Loads one train's trip and keeps it fresh: every 30 s while the page is visible, at once when
// the page becomes visible again after a while, and never in a background tab (no NS load for
// pages nobody looks at). A failed refresh keeps the last good trip on screen.
import { useCallback, useEffect, useRef, useState } from 'react';
import { api as defaultClient, ApiError, errorKey, type Client } from '../api/client';
import type { Trip } from '../api/types';
import type { MessageKey } from '../i18n/messages';

export const refreshMs = 30_000;

export type TripState =
  | { status: 'loading' }
  | { status: 'notFound' }
  | { status: 'failed'; message: MessageKey }
  | { status: 'ready'; trip: Trip; refreshing: boolean; refreshFailed: boolean };

export function useTrip(train: string, client: Pick<Client, 'trip'> = defaultClient) {
  const [state, setState] = useState<TripState>({ status: 'loading' });
  const [now, setNow] = useState(() => Date.now());
  const lastLoad = useRef(0);
  const inFlight = useRef<AbortController | null>(null);
  // The client in a ref: a new client object (such as an inline one) must not restart the loop.
  const clientRef = useRef(client);
  useEffect(() => {
    clientRef.current = client;
  });

  const load = useCallback(async () => {
    inFlight.current?.abort();
    const ctl = new AbortController();
    inFlight.current = ctl;
    lastLoad.current = Date.now();
    setState((s) => (s.status === 'ready' ? { ...s, refreshing: true } : s));
    try {
      const trip = await clientRef.current.trip(train, ctl.signal);
      if (ctl.signal.aborted) return;
      setNow(Date.now());
      setState({ status: 'ready', trip, refreshing: false, refreshFailed: false });
    } catch (err) {
      if (ctl.signal.aborted) return;
      setNow(Date.now());
      setState((s) => {
        if (s.status === 'ready') return { ...s, refreshing: false, refreshFailed: true };
        if (err instanceof ApiError && err.status === 404) return { status: 'notFound' };
        return { status: 'failed', message: errorKey(err) };
      });
    }
  }, [train]);

  useEffect(() => {
    setState({ status: 'loading' });
    void load();
    const tick = setInterval(() => {
      if (document.visibilityState === 'visible') void load();
    }, refreshMs);
    const onVisible = () => {
      if (document.visibilityState === 'visible' && Date.now() - lastLoad.current >= refreshMs) void load();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(tick);
      document.removeEventListener('visibilitychange', onVisible);
      inFlight.current?.abort();
    };
  }, [load]);

  return { state, now, reload: load };
}
