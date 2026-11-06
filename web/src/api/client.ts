// The API client. Every call goes to the same origin (/api, proxied by nginx). Transient
// failures (no connection, 502, 503, 504) are retried twice with backoff; a 429 or any other
// client error is not, so a busy backend is not pushed harder.
import type { CarriageResult, Departures, Fix, MatchResult, NearestStations, Trip } from './types';

// ApiError carries the backend's error code ({"error": "..."}), or "network" / "timeout" when
// there was no answer, so the page can say what happened in the user's language.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  constructor(status: number, code: string) {
    super(`api: ${status} ${code}`);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

type Options = {
  fetch?: typeof fetch;
  sleep?: (ms: number) => Promise<void>;
  retries?: number;
  backoffMs?: number;
  timeoutMs?: number;
};

const transient = new Set([502, 503, 504]);

export function createClient(opts: Options = {}) {
  const doFetch = opts.fetch ?? ((...a: Parameters<typeof fetch>) => fetch(...a));
  const sleep = opts.sleep ?? ((ms: number) => new Promise<void>((r) => setTimeout(r, ms)));
  const retries = opts.retries ?? 2;
  const backoffMs = opts.backoffMs ?? 500;
  const timeoutMs = opts.timeoutMs ?? 15000;

  async function once<T>(path: string, init: RequestInit, signal?: AbortSignal): Promise<T> {
    const timeout = AbortSignal.timeout(timeoutMs);
    const sig = signal ? AbortSignal.any([signal, timeout]) : timeout;
    let res: Response;
    try {
      res = await doFetch(path, { ...init, signal: sig, headers: { Accept: 'application/json', ...init.headers } });
    } catch (err) {
      if (signal?.aborted) throw err; // the caller gave up: not an error to show
      throw new ApiError(0, timeout.aborted ? 'timeout' : 'network');
    }
    if (!res.ok) {
      let code = 'unknown';
      try {
        const body = (await res.json()) as { error?: unknown };
        if (typeof body.error === 'string') code = body.error;
      } catch {
        // not JSON (a proxy error page)
      }
      throw new ApiError(res.status, code);
    }
    return (await res.json()) as T;
  }

  async function request<T>(path: string, init: RequestInit = {}, signal?: AbortSignal): Promise<T> {
    let wait = backoffMs;
    for (let attempt = 0; ; attempt++) {
      try {
        return await once<T>(path, init, signal);
      } catch (err) {
        const retry = err instanceof ApiError && (err.status === 0 || transient.has(err.status));
        if (!retry || attempt >= retries || signal?.aborted) throw err;
        await sleep(wait);
        wait *= 2;
      }
    }
  }

  const post = (body: unknown): RequestInit => ({
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

  return {
    trip: (train: string, signal?: AbortSignal) => request<Trip>(`/api/trains/${encodeURIComponent(train)}`, {}, signal),
    departures: (station: string, signal?: AbortSignal) =>
      request<Departures>(`/api/stations/${encodeURIComponent(station)}/departures`, {}, signal),
    match: (fixes: Fix[], signal?: AbortSignal) => request<MatchResult>('/api/match', post({ fixes }), signal),
    carriage: (carriage: string, fixes: Fix[], signal?: AbortSignal) =>
      request<CarriageResult>('/api/carriage', post(fixes.length ? { carriage, fixes } : { carriage }), signal),
    nearestStations: (fixes: Fix[], signal?: AbortSignal) => request<NearestStations>('/api/stations/nearest', post({ fixes }), signal),
  };
}

export type Client = ReturnType<typeof createClient>;

export const api = createClient();

// errorKey maps an error to the dictionary key of the message to show.
export function errorKey(err: unknown) {
  if (!(err instanceof ApiError)) return 'error.unknown' as const;
  if (err.status === 0) return 'error.network' as const;
  if (err.status === 429) return 'error.rateLimited' as const;
  if (err.status === 404) return 'error.notFound' as const;
  if (err.status === 400 || err.status === 413) return 'error.invalid' as const;
  if (err.status >= 500) return 'error.unavailable' as const;
  return 'error.unknown' as const;
}
