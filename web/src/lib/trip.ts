// Pure helpers for the trip view: where the train is on its route, and how to show times,
// delays and platforms. "now" is passed in, so they are easy to test.
import type { Stop, TripEvent } from '../api/types';

export type Progress = {
  state: 'before' | 'running' | 'arrived';
  past: Stop[]; // stops already left, oldest first
  next: Stop | null; // the next stop (or the origin, before departure)
  later: Stop[]; // the stops after next
};

// A train counts as gone from a stop this long after its expected departure, so a train that is
// a few seconds late in the data is not shown as left already.
const graceMs = 30_000;

export function expected(e?: TripEvent): number | null {
  const t = e?.actual ?? e?.planned;
  return t ? Date.parse(t) : null;
}

// leftAt is when the train is expected to leave (or, at the last stop, arrive at) a stop.
function leftAt(s: Stop): number | null {
  return expected(s.departure) ?? expected(s.arrival);
}

export function progress(stops: readonly Stop[], now: number): Progress {
  const halts = stops.filter((s) => s.status !== 'PASSING');
  const i = halts.findIndex((s) => {
    const t = leftAt(s);
    return t === null || t + graceMs > now;
  });
  if (i === -1) return { state: 'arrived', past: halts.slice(0, -1), next: halts.at(-1) ?? null, later: [] };
  return { state: i === 0 ? 'before' : 'running', past: halts.slice(0, i), next: halts[i] ?? null, later: halts.slice(i + 1) };
}

export function delayMinutes(e?: TripEvent): number {
  return e ? Math.round(e.delaySeconds / 60) : 0;
}

export function platform(e?: TripEvent): { track: string | null; changed: boolean; planned: string | null } {
  const planned = e?.plannedTrack || null;
  const actual = e?.actualTrack || null;
  return { track: actual ?? planned, changed: !!actual && !!planned && actual !== planned, planned };
}

// The event that matters for a stop: departure, except at the last stop.
export function mainEvent(s: Stop): TripEvent | undefined {
  return s.departure ?? s.arrival;
}

export function cancelled(s: Stop): boolean {
  return !!(s.arrival?.cancelled || s.departure?.cancelled);
}

// Times are shown as the station clock shows them: Dutch time, whatever the phone's zone.
const clock = new Intl.DateTimeFormat('nl-NL', { hour: '2-digit', minute: '2-digit', timeZone: 'Europe/Amsterdam' });

export function hhmm(iso?: string | null): string {
  if (!iso) return '';
  const t = Date.parse(iso);
  return Number.isNaN(t) ? '' : clock.format(t);
}

// The trip's delay: the delay at the next stop, the figure travellers care about.
export function currentDelay(p: Progress): number {
  return p.next ? delayMinutes(p.next.arrival ?? p.next.departure) : 0;
}

export type CrowdLevel = 0 | 1 | 2 | 3; // unknown, low, medium, high

export function crowdLevel(crowd?: string): CrowdLevel {
  return crowd === 'LOW' ? 1 : crowd === 'MEDIUM' ? 2 : crowd === 'HIGH' ? 3 : 0;
}
