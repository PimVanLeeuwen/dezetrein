// The browser's Geolocation API, wrapped so the flow can be tested with a fake. Readings are
// passed on as they come and never stored here.
import type { Fix } from '../api/types';

// The area the backend accepts (the NS network with its border routes).
const area = { minLat: 49, maxLat: 55, minLng: 2, maxLng: 11 };
// The backend's limit on reported accuracy, in metres.
export const maxAccuracyM = 20000;

export type GeoError = 'denied' | 'unavailable' | 'unsupported';
export type Permission = 'granted' | 'prompt' | 'denied' | 'unknown';

export type Geo = {
  // watch reports every reading until the returned function is called.
  watch(onFix: (fix: Fix) => void, onError: (err: GeoError) => void): () => void;
  permission(): Promise<Permission>;
};

// toFix converts a browser reading. Speed and heading are null when the phone does not report
// them (the API gives null, or NaN for the heading while standing still).
export function toFix(pos: GeolocationPosition): Fix {
  const num = (v: number | null) => (v === null || Number.isNaN(v) ? null : v);
  const { latitude, longitude, accuracy, speed, heading } = pos.coords;
  return { lat: latitude, lng: longitude, accuracy: Math.max(1, accuracy), speed: num(speed), heading: num(heading), t: Math.round(pos.timestamp) };
}

export function inArea(f: Fix): boolean {
  return f.lat >= area.minLat && f.lat <= area.maxLat && f.lng >= area.minLng && f.lng <= area.maxLng;
}

// spread picks up to n readings evenly over the list, always including the newest, as the
// matcher works best with readings spread over time.
export function spread<T>(list: readonly T[], n: number): T[] {
  if (list.length <= n) return [...list];
  return Array.from({ length: n }, (_, i) => list[Math.round(((list.length - 1) * i) / (n - 1))]!);
}

export const browserGeo: Geo = {
  watch(onFix, onError) {
    if (!('geolocation' in navigator)) {
      onError('unsupported');
      return () => {};
    }
    const id = navigator.geolocation.watchPosition(
      (pos) => onFix(toFix(pos)),
      (err) => onError(err.code === err.PERMISSION_DENIED ? 'denied' : 'unavailable'),
      // High accuracy where the phone can, but any reading is welcome: inside a train it is
      // often only a cell or Wi-Fi position.
      { enableHighAccuracy: true, maximumAge: 5000, timeout: 20000 },
    );
    return () => navigator.geolocation.clearWatch(id);
  },
  async permission() {
    try {
      const status = await navigator.permissions?.query({ name: 'geolocation' });
      return status?.state ?? 'unknown';
    } catch {
      return 'unknown';
    }
  },
};
