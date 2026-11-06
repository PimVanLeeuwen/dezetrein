import { vi } from 'vitest';
import type { Fix, MatchResult } from '../api/types';
import type { Geo, GeoError, Permission } from '../lib/geo';

// A controllable Geolocation API: the test pushes readings and errors.
export function fakeGeo(permission: Permission = 'prompt') {
  let onFix: ((f: Fix) => void) | null = null;
  let onError: ((e: GeoError) => void) | null = null;
  const stop = vi.fn(() => {
    onFix = null;
    onError = null;
  });
  const geo: Geo = {
    watch: vi.fn((f, e) => {
      onFix = f;
      onError = e;
      return stop;
    }),
    permission: vi.fn(async () => permission),
  };
  return {
    geo,
    stop,
    watching: () => onFix !== null,
    fix: (over: Partial<Fix> = {}) => onFix?.({ lat: 52.09, lng: 5.11, accuracy: 30, speed: null, heading: null, t: Date.now(), ...over }),
    fail: (e: GeoError) => onError?.(e),
  };
}

export const candidate = (train: string, destination?: string, carriages: string[] = []) => ({ train, category: 'IC', destination, carriages });

export const result = (confidence: MatchResult['confidence'], candidates: MatchResult['candidates']): MatchResult => ({
  match: confidence === 'high' ? (candidates[0]?.train ?? null) : null,
  confidence,
  candidates,
  as_of: '2026-11-06T12:00:00Z',
  stale: false,
});
