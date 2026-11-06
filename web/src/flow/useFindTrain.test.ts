import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import type { Fix, MatchResult } from '../api/types';
import { candidate, fakeGeo, result } from '../test/fakes';
import { timing, useFindTrain } from './useFindTrain';

beforeEach(() => vi.useFakeTimers({ now: new Date('2026-11-06T12:00:00Z') }));
afterEach(() => vi.useRealTimers());

function setup(answers: (MatchResult | Error)[], permission: 'prompt' | 'granted' = 'prompt') {
  const g = fakeGeo(permission);
  const sent: Fix[][] = [];
  const client = {
    match: vi.fn(async (fixes: Fix[]) => {
      sent.push(fixes);
      const a = answers[Math.min(sent.length - 1, answers.length - 1)]!;
      if (a instanceof Error) throw a;
      return a;
    }),
  };
  const hook = renderHook(() => useFindTrain({ geo: g.geo, client }));
  return { ...g, client, sent, hook };
}

const advance = (ms: number) => act(async () => void (await vi.advanceTimersByTimeAsync(ms)));

describe('useFindTrain', () => {
  it('answers early when the match is clear, and stops locating', async () => {
    const s = setup([result('high', [candidate('3539', 'Venlo', ['9574'])])]);
    expect(s.hook.result.current.state.phase).toBe('intro');
    act(() => s.hook.result.current.start());
    act(() => s.fix());
    expect(s.hook.result.current.state).toMatchObject({ phase: 'locating', readings: 1, accuracyM: 30 });
    await advance(timing.firstTryMs);
    expect(s.hook.result.current.state).toMatchObject({ phase: 'confirm', result: { confidence: 'high' } });
    expect(s.stop).toHaveBeenCalled();
    await advance(timing.giveUpMs);
    expect(s.client.match).toHaveBeenCalledTimes(1);
  });

  it('keeps collecting when unsure, then asks with five readings spread over the window', async () => {
    const s = setup([result('low', [candidate('1'), candidate('2')]), result('low', [candidate('1'), candidate('2')])]);
    act(() => s.hook.result.current.start());
    for (let i = 0; i < 8; i++) {
      act(() => s.fix({ t: Date.now() }));
      await advance(1000);
    }
    expect(s.hook.result.current.state.phase).toBe('locating');
    await advance(timing.finalMs);
    expect(s.hook.result.current.state).toMatchObject({ phase: 'confirm', result: { confidence: 'low' } });
    expect(s.sent).toHaveLength(2);
    expect(s.sent[1]).toHaveLength(5);
  });

  it('reports no train when the final answer has no candidates', async () => {
    const s = setup([result('none', [])]);
    act(() => s.hook.result.current.start());
    act(() => s.fix());
    await advance(timing.finalMs);
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'none' });
  });

  it('gives up without any reading', async () => {
    const s = setup([result('high', [candidate('1')])]);
    act(() => s.hook.result.current.start());
    act(() => s.fail('unavailable')); // briefly unavailable: not fatal
    await advance(timing.finalMs);
    expect(s.hook.result.current.state.phase).toBe('locating');
    await advance(timing.giveUpMs - timing.finalMs);
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'timeout' });
    expect(s.client.match).not.toHaveBeenCalled();
  });

  it('asks as soon as a late first reading arrives after the window', async () => {
    const s = setup([result('high', [candidate('1')])]);
    act(() => s.hook.result.current.start());
    await advance(timing.finalMs + 1000);
    act(() => s.fix());
    await advance(0);
    expect(s.hook.result.current.state.phase).toBe('confirm');
  });

  it('stops at once when location is denied or outside the network', async () => {
    const s = setup([]);
    act(() => s.hook.result.current.start());
    act(() => s.fail('denied'));
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'denied' });
    act(() => s.hook.result.current.start());
    act(() => s.fix({ lat: 40.4, lng: -3.7 }));
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'outside' });
    expect(s.watching()).toBe(false);
  });

  it('retries a failed early request, and reports a failed final one', async () => {
    const s = setup([new ApiError(503, 'unavailable'), new ApiError(0, 'network')]);
    act(() => s.hook.result.current.start());
    act(() => s.fix());
    await advance(timing.firstTryMs);
    expect(s.hook.result.current.state.phase).toBe('locating');
    await advance(timing.finalMs);
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'api', message: 'error.network' });
  });

  it('stops at once on a rate limit', async () => {
    const s = setup([new ApiError(429, 'rate_limited')]);
    act(() => s.hook.result.current.start());
    act(() => s.fix());
    await advance(timing.firstTryMs);
    expect(s.hook.result.current.state).toEqual({ phase: 'failed', reason: 'api', message: 'error.rateLimited' });
  });

  it('ignores readings that are too coarse for the backend', async () => {
    const s = setup([result('high', [candidate('1')])]);
    act(() => s.hook.result.current.start());
    act(() => s.fix({ accuracy: 50000 }));
    expect(s.hook.result.current.state).toMatchObject({ readings: 0 });
  });

  it('forgets everything on cancel: a late answer does not come back', async () => {
    let release: (r: MatchResult) => void = () => {};
    const g = fakeGeo();
    const client = { match: vi.fn<(fixes: Fix[]) => Promise<MatchResult>>(() => new Promise((r) => (release = r))) };
    const { result: h } = renderHook(() => useFindTrain({ geo: g.geo, client }));
    act(() => h.current.start());
    act(() => g.fix());
    await advance(timing.firstTryMs);
    act(() => h.current.reset());
    await act(async () => release(result('high', [candidate('1')])));
    expect(h.current.state).toEqual({ phase: 'intro' });
    expect(g.watching()).toBe(false);
    // A new attempt starts from scratch: only its own readings are sent.
    client.match.mockResolvedValue(result('high', [candidate('2')]));
    act(() => h.current.start());
    act(() => g.fix({ lat: 52.1 }));
    await advance(timing.firstTryMs);
    expect(client.match.mock.calls[1]![0]).toHaveLength(1);
    expect(client.match.mock.calls[1]![0][0]!.lat).toBe(52.1);
  });

  it('starts by itself when location was already allowed', async () => {
    const s = setup([result('high', [candidate('1')])], 'granted');
    await advance(0);
    expect(s.hook.result.current.state.phase).toBe('locating');
    expect(s.geo.watch).toHaveBeenCalled();
  });

  it('stops watching when the page closes', async () => {
    const s = setup([]);
    act(() => s.hook.result.current.start());
    s.hook.unmount();
    expect(s.watching()).toBe(false);
  });
});
