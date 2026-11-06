import { describe, expect, it } from 'vitest';
import { inArea, spread, toFix } from './geo';

const pos = (c: Partial<GeolocationCoordinates>, timestamp = 1793966000123.4) =>
  ({ coords: { latitude: 52.09, longitude: 5.11, accuracy: 30, speed: null, heading: null, altitude: null, altitudeAccuracy: null, ...c }, timestamp }) as GeolocationPosition;

describe('toFix', () => {
  it('keeps speed and heading only when reported', () => {
    expect(toFix(pos({ speed: 33.2, heading: 41 }))).toEqual({ lat: 52.09, lng: 5.11, accuracy: 30, speed: 33.2, heading: 41, t: 1793966000123 });
    expect(toFix(pos({ speed: 0, heading: NaN }))).toMatchObject({ speed: 0, heading: null });
    expect(toFix(pos({ accuracy: 0 })).accuracy).toBe(1);
  });
});

describe('inArea', () => {
  it.each([
    [52.09, 5.11, true],
    [50.85, 4.35, true], // Brussels, a border route
    [40.4, -3.7, false],
    [52.52, 13.4, false], // Berlin
  ])('%f, %f: %s', (lat, lng, want) => expect(inArea({ lat, lng, accuracy: 10, speed: null, heading: null, t: 1 })).toBe(want));
});

describe('spread', () => {
  it('picks evenly and keeps the first and the newest', () => {
    expect(spread([1, 2, 3], 5)).toEqual([1, 2, 3]);
    expect(spread([0, 1, 2, 3, 4, 5, 6, 7, 8], 5)).toEqual([0, 2, 4, 6, 8]);
    expect(spread([0, 1, 2, 3, 4, 5], 5)).toEqual([0, 1, 3, 4, 5]);
  });
});

describe('browserGeo', () => {
  it('watches positions, maps errors and clears the watch', async () => {
    const { browserGeo } = await import('./geo');
    const { vi } = await import('vitest');
    let ok: PositionCallback = () => {};
    let fail: PositionErrorCallback = () => {};
    const clearWatch = vi.fn();
    vi.stubGlobal('navigator', {
      ...navigator,
      geolocation: { watchPosition: (s: PositionCallback, e: PositionErrorCallback) => ((ok = s), (fail = e), 7), clearWatch },
      permissions: { query: async () => ({ state: 'granted' }) },
    });
    const fixes: unknown[] = [];
    const errors: string[] = [];
    const stop = browserGeo.watch((f) => fixes.push(f), (e) => errors.push(e));
    ok(pos({ accuracy: 1414 }));
    fail({ code: 1, PERMISSION_DENIED: 1 } as GeolocationPositionError);
    fail({ code: 3, PERMISSION_DENIED: 1 } as GeolocationPositionError);
    stop();
    expect(fixes).toHaveLength(1);
    expect(errors).toEqual(['denied', 'unavailable']);
    expect(clearWatch).toHaveBeenCalledWith(7);
    expect(await browserGeo.permission()).toBe('granted');
    vi.unstubAllGlobals();
  });

  it('copes with a browser without geolocation or permissions', async () => {
    const { browserGeo } = await import('./geo');
    const { vi } = await import('vitest');
    vi.stubGlobal('navigator', { languages: ['nl'] });
    const errors: string[] = [];
    browserGeo.watch(() => {}, (e) => errors.push(e))();
    expect(errors).toEqual(['unsupported']);
    expect(await browserGeo.permission()).toBe('unknown');
    vi.unstubAllGlobals();
  });
});

describe('sample', () => {
  it('resolves with the readings it wanted, then stops watching', async () => {
    const { fakeGeo } = await import('../test/fakes');
    const g = fakeGeo();
    const p = (await import('./geo')).sample(g.geo, { want: 2, maxMs: 5000 });
    g.fix({ lat: 52.1 });
    g.fix({ accuracy: 99999 }); // too coarse: ignored
    g.fix({ lat: 52.2 });
    const got = await p;
    expect(got.map((f) => f.lat)).toEqual([52.1, 52.2]);
    expect(g.watching()).toBe(false);
  });

  it('resolves with what it has after the time limit, and rejects with nothing', async () => {
    const { vi } = await import('vitest');
    const { fakeGeo } = await import('../test/fakes');
    const { sample } = await import('./geo');
    vi.useFakeTimers();
    const a = fakeGeo();
    const one = sample(a.geo, { want: 3, maxMs: 1000 });
    a.fix();
    vi.advanceTimersByTime(1000);
    expect(await one).toHaveLength(1);
    const b = fakeGeo();
    const none = sample(b.geo, { maxMs: 1000 });
    vi.advanceTimersByTime(1000);
    await expect(none).rejects.toThrow('timeout');
    vi.useRealTimers();
  });

  it('rejects on denial, outside the network, and abort; copes with a synchronous error', async () => {
    const { fakeGeo } = await import('../test/fakes');
    const { sample } = await import('./geo');
    const a = fakeGeo();
    const denied = sample(a.geo);
    a.fail('denied');
    await expect(denied).rejects.toThrow('denied');
    const b = fakeGeo();
    const outside = sample(b.geo);
    b.fix({ lat: 40, lng: -3 });
    await expect(outside).rejects.toThrow('outside');
    const c = fakeGeo();
    const ctl = new AbortController();
    const aborted = sample(c.geo, { signal: ctl.signal });
    ctl.abort();
    await expect(aborted).rejects.toThrow('aborted');
    expect(c.watching()).toBe(false);
    const stop = () => {};
    const sync = { watch: (_: unknown, onError: (e: 'unsupported') => void) => (onError('unsupported'), stop), permission: async () => 'unknown' as const };
    await expect(sample(sync)).rejects.toThrow('unsupported');
  });
});
