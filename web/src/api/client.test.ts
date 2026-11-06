import { describe, expect, it, vi } from 'vitest';
import { ApiError, createClient, errorKey } from './client';

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function setup(...responses: (Response | Error)[]) {
  const calls: { url: string; init: RequestInit }[] = [];
  const waits: number[] = [];
  const fetch = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    const r = responses[Math.min(calls.length - 1, responses.length - 1)]!;
    if (r instanceof Error) throw r;
    return r.clone();
  });
  const client = createClient({ fetch, sleep: async (ms) => void waits.push(ms) });
  return { client, calls, waits };
}

describe('api client', () => {
  it('retries gateway errors with backoff, then succeeds', async () => {
    const { client, calls, waits } = setup(json(503, { error: 'unavailable' }), json(502, {}), json(200, { train: '2450' }));
    await expect(client.trip('2450')).resolves.toEqual({ train: '2450' });
    expect(calls).toHaveLength(3);
    expect(waits).toEqual([500, 1000]);
    expect(calls[0]!.url).toBe('/api/trains/2450');
  });

  it('gives up after two retries with the backend error code', async () => {
    const { client, calls } = setup(json(503, { error: 'unavailable' }));
    await expect(client.trip('2450')).rejects.toMatchObject({ status: 503, code: 'unavailable' });
    expect(calls).toHaveLength(3);
  });

  it.each([429, 404, 400, 500])('does not retry %i', async (status) => {
    const { client, calls } = setup(json(status, { error: 'x' }));
    await expect(client.trip('1')).rejects.toBeInstanceOf(ApiError);
    expect(calls).toHaveLength(1);
  });

  it('retries a lost connection and reports it as network', async () => {
    const { client, calls } = setup(new TypeError('Failed to fetch'));
    await expect(client.trip('1')).rejects.toMatchObject({ status: 0, code: 'network' });
    expect(calls).toHaveLength(3);
  });

  it('copes with an error page that is not JSON', async () => {
    const { client } = setup(new Response('<html>Bad gateway</html>', { status: 404 }));
    await expect(client.trip('1')).rejects.toMatchObject({ status: 404, code: 'unknown' });
  });

  it('does not retry when the caller aborted', async () => {
    const ctl = new AbortController();
    const { client, calls } = setup(new DOMException('aborted', 'AbortError'));
    ctl.abort();
    await expect(client.trip('1', ctl.signal)).rejects.toThrow();
    expect(calls).toHaveLength(1);
  });

  it('posts fixes as JSON, and leaves them out of a carriage lookup without location', async () => {
    const { client, calls } = setup(json(200, {}));
    const fix = { lat: 52.09, lng: 5.11, accuracy: 30, speed: null, heading: null, t: 1 };
    await client.match([fix]);
    await client.carriage('9402', []);
    await client.carriage('9402', [fix]);
    await client.departures('UT');
    expect(calls[0]!.url).toBe('/api/match');
    expect(calls[0]!.init.method).toBe('POST');
    expect(JSON.parse(String(calls[0]!.init.body))).toEqual({ fixes: [fix] });
    expect(JSON.parse(String(calls[1]!.init.body))).toEqual({ carriage: '9402' });
    expect(JSON.parse(String(calls[2]!.init.body))).toEqual({ carriage: '9402', fixes: [fix] });
    expect(calls[3]!.url).toBe('/api/stations/UT/departures');
    // Location goes in the body only, never in the URL.
    for (const c of calls) expect(c.url).not.toContain('52.09');
  });

  it('maps errors to messages', () => {
    expect(errorKey(new ApiError(0, 'network'))).toBe('error.network');
    expect(errorKey(new ApiError(429, 'rate_limited'))).toBe('error.rateLimited');
    expect(errorKey(new ApiError(404, 'not_found'))).toBe('error.notFound');
    expect(errorKey(new ApiError(400, 'invalid_fixes'))).toBe('error.invalid');
    expect(errorKey(new ApiError(503, 'unavailable'))).toBe('error.unavailable');
    expect(errorKey(new Error('x'))).toBe('error.unknown');
  });
});
