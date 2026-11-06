import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import type { Trip } from '../api/types';
import { tripFixture } from '../test/trip';
import { refreshMs, useTrip } from './useTrip';

let visibility: DocumentVisibilityState = 'visible';
beforeEach(() => {
  vi.useFakeTimers();
  visibility = 'visible';
  vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() => visibility);
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

const flush = () => act(async () => void (await vi.advanceTimersByTimeAsync(0)));
const advance = (ms: number) => act(async () => void (await vi.advanceTimersByTimeAsync(ms)));

function setup(...answers: (Trip | Error)[]) {
  const trip = vi.fn(async () => {
    const a = answers[Math.min(trip.mock.calls.length - 1, answers.length - 1)]!;
    if (a instanceof Error) throw a;
    return a;
  });
  const hook = renderHook(() => useTrip('2450', { trip }));
  return { trip, hook };
}

describe('useTrip', () => {
  it('loads the trip', async () => {
    const { hook, trip } = setup(tripFixture);
    expect(hook.result.current.state.status).toBe('loading');
    await flush();
    expect(hook.result.current.state).toMatchObject({ status: 'ready', trip: { train: '2450' } });
    expect(trip).toHaveBeenCalledWith('2450', expect.any(AbortSignal));
  });

  it('reports an unknown train, and other errors', async () => {
    const a = setup(new ApiError(404, 'not_found'));
    await flush();
    expect(a.hook.result.current.state).toEqual({ status: 'notFound' });
    a.hook.unmount();
    const b = setup(new ApiError(503, 'unavailable'));
    await flush();
    expect(b.hook.result.current.state).toEqual({ status: 'failed', message: 'error.unavailable' });
  });

  it('refreshes every 30 s while visible, and not in the background', async () => {
    const { trip } = setup(tripFixture);
    await flush();
    await advance(refreshMs);
    expect(trip).toHaveBeenCalledTimes(2);
    visibility = 'hidden';
    await advance(refreshMs * 3);
    expect(trip).toHaveBeenCalledTimes(2);
    visibility = 'visible';
    await act(async () => void document.dispatchEvent(new Event('visibilitychange')));
    await flush();
    expect(trip).toHaveBeenCalledTimes(3); // back after a while: refreshed at once
  });

  it('does not refresh on a quick tab switch', async () => {
    const { trip } = setup(tripFixture);
    await flush();
    await advance(5000);
    await act(async () => void document.dispatchEvent(new Event('visibilitychange')));
    expect(trip).toHaveBeenCalledTimes(1);
  });

  it('keeps the last trip when a refresh fails, and recovers', async () => {
    const { hook } = setup(tripFixture, new ApiError(0, 'network'), { ...tripFixture, destination: 'Breda' });
    await flush();
    await advance(refreshMs);
    expect(hook.result.current.state).toMatchObject({ status: 'ready', refreshFailed: true, trip: { destination: 'Rotterdam Centraal' } });
    await advance(refreshMs);
    expect(hook.result.current.state).toMatchObject({ status: 'ready', refreshFailed: false, trip: { destination: 'Breda' } });
  });

  it('stops refreshing when the page closes', async () => {
    const { hook, trip } = setup(tripFixture);
    await flush();
    hook.unmount();
    await advance(refreshMs * 2);
    expect(trip).toHaveBeenCalledTimes(1);
  });
});
