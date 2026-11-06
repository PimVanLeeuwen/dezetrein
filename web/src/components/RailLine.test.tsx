import { render } from '@testing-library/react';
import { useRef } from 'react';
import { act } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { measure, RailLine } from './RailLine';

describe('measure', () => {
  it('runs above the first stop, down the right edge, past the last', () => {
    const g = measure(400, 300, [
      { top: 100, bottom: 180, right: 340 },
      { top: 200, bottom: 280, right: 340 },
    ])!;
    expect(g.x).toBe(378);
    expect(g.d).toBe('M -48 56 H 332 Q 378 56 378 102 V 344');
    expect(g.stations).toEqual([140, 240]);
    expect(g.spurs).toEqual([
      [340, 140],
      [340, 240],
    ]);
    expect(g.end).toBe(344);
    expect(g.height).toBe(364);
  });

  it('starts at the left edge of the window when the content is centred', () => {
    const g = measure(400, 300, [{ top: 100, bottom: 180, right: 340 }], 300)!;
    expect(g.d.startsWith('M -308 56 ')).toBe(true);
  });

  it('draws nothing without room or stops', () => {
    expect(measure(0, 100, [{ top: 0, bottom: 10, right: 5 }])).toBeNull();
    expect(measure(400, 100, [])).toBeNull();
  });
});

function Harness() {
  const box = useRef<HTMLDivElement>(null);
  const a = useRef<HTMLButtonElement>(null);
  return (
    <div ref={box}>
      <RailLine container={box} stops={[a]} />
      <button ref={a} type="button">
        stop
      </button>
    </div>
  );
}

describe('RailLine', () => {
  it('renders nothing in a layout without size (jsdom), and is hidden from assistive tech', () => {
    const { queryByTestId } = render(<Harness />);
    expect(queryByTestId('rail-line')).toBeNull();
  });
});

describe('RailLine in a real layout', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('draws a station and a spur per stop, hidden from assistive tech, and redraws on resize', async () => {
    let width = 400;
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      const r = (x: number, y: number, w: number, h: number) => ({ x, y, left: x, top: y, width: w, height: h, right: x + w, bottom: y + h, toJSON: () => ({}) }) as DOMRect;
      return this.tagName === 'BUTTON' ? r(20, 100, width - 80, 80) : r(20, 0, width, 300);
    });
    const observers: { cb: () => void; disconnect: () => void }[] = [];
    vi.stubGlobal(
      'ResizeObserver',
      class {
        cb: () => void;
        disconnect = vi.fn();
        constructor(cb: () => void) {
          this.cb = cb;
          observers.push(this);
        }
        observe() {}
      },
    );
    const { getByTestId, unmount } = render(<Harness />);
    await act(async () => {});
    const svg = getByTestId('rail-line');
    expect(svg).toHaveAttribute('aria-hidden', 'true');
    expect(svg.querySelectorAll('circle')).toHaveLength(3); // one station, the terminus, the train
    expect(svg.querySelectorAll('line')).toHaveLength(1);
    expect(svg.getAttribute('width')).toBe('400');

    width = 320;
    await act(async () => observers.forEach((o) => o.cb()));
    expect(getByTestId('rail-line').getAttribute('width')).toBe('320');
    unmount();
    expect(observers[0]!.disconnect).toHaveBeenCalled();
  });
});
