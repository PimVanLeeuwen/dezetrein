import { describe, expect, it } from 'vitest';
import { at, tripFixture } from '../test/trip';
import { cancelled, crowdLevel, currentDelay, delayMinutes, hhmm, platform, progress } from './trip';

const names = (s: { name: string }[]) => s.map((x) => x.name);

describe('progress', () => {
  it('before departure: the origin is next', () => {
    const p = progress(tripFixture.stops, at.before);
    expect(p.state).toBe('before');
    expect(p.next?.name).toBe('Lelystad Centrum');
    expect(p.past).toEqual([]);
    expect(names(p.later)).not.toContain('Almere Oostvaarders'); // passing stations are left out
  });

  it('on the way: the stops behind, the next one and the rest', () => {
    const p = progress(tripFixture.stops, at.betweenZuidAndSchiphol);
    expect(p.state).toBe('running');
    expect(names(p.past)).toEqual(['Lelystad Centrum', 'Almere Buiten', 'Amsterdam Zuid']);
    expect(p.next?.name).toBe('Schiphol Airport');
    expect(names(p.later)).toEqual(['Rotterdam Centraal']);
    expect(currentDelay(p)).toBe(16);
  });

  it('uses the expected time with a short grace, not the planned time', () => {
    // 15:49:54 local is the expected departure from Amsterdam Zuid; 15:45 planned would be long gone.
    const justAfter = Date.parse('2026-11-04T14:50:10Z');
    expect(progress(tripFixture.stops, justAfter).next?.name).toBe('Amsterdam Zuid');
    expect(progress(tripFixture.stops, justAfter + 30_000).next?.name).toBe('Schiphol Airport');
  });

  it('after arrival: the destination stays as the last stop', () => {
    const p = progress(tripFixture.stops, at.arrived);
    expect(p.state).toBe('arrived');
    expect(p.next?.name).toBe('Rotterdam Centraal');
    expect(p.later).toEqual([]);
  });
});

describe('stop details', () => {
  const shl = tripFixture.stops[4]!;
  it('reads delays and platform changes', () => {
    expect(delayMinutes(shl.departure)).toBe(16);
    expect(delayMinutes(undefined)).toBe(0);
    expect(platform(shl.departure)).toEqual({ track: '6', changed: true, planned: '5-6' });
    expect(platform(shl.arrival)).toEqual({ track: '6', changed: false, planned: '6' });
    expect(platform(undefined)).toEqual({ track: null, changed: false, planned: null });
  });

  it('shows times in Dutch time, whatever the device zone', () => {
    expect(hhmm('2026-11-04T14:57:53Z')).toBe('15:57');
    expect(hhmm('2026-07-01T12:00:00Z')).toBe('14:00'); // summer time
    expect(hhmm(undefined)).toBe('');
    expect(hhmm('nonsense')).toBe('');
  });

  it('maps crowding and cancellations', () => {
    expect([crowdLevel('LOW'), crowdLevel('MEDIUM'), crowdLevel('HIGH'), crowdLevel('UNKNOWN'), crowdLevel()]).toEqual([1, 2, 3, 0, 0]);
    expect(cancelled(shl)).toBe(false);
    expect(cancelled({ ...shl, arrival: { ...shl.arrival!, cancelled: true } })).toBe(true);
  });
});
