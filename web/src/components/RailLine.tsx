import { useEffect, useState, type RefObject } from 'react';

// RailLine draws the site's motif: a thick rail line that comes in from the left above the first
// stop, bends down along the right edge past every stop (each gets a station and a short spur),
// and ends at a terminus. A train dot rides along it. The line is measured from the real layout,
// so it fits any width, font and language. Decorative only: hidden from assistive technology.

type Props = {
  container: RefObject<HTMLElement | null>;
  stops: RefObject<HTMLElement | null>[];
};

type Geometry = { width: number; height: number; d: string; x: number; stations: number[]; spurs: [number, number][]; end: number };

// Where the vertical line runs, from the container's right edge, and how far it extends past
// the stops above and below.
const inset = 22;
const radius = 46;
const above = 44;
const below = 64;
const minBleed = 48; // the line enters from beyond the left edge of the window

// left is the container's distance from the window's left edge, so the line can start there.
export function measure(width: number, height: number, stops: { top: number; bottom: number; right: number }[], left = 0): Geometry | null {
  if (!width || stops.length === 0) return null;
  const x = width - inset;
  const first = stops[0]!;
  const last = stops[stops.length - 1]!;
  const y0 = first.top - above;
  const end = last.bottom + below;
  const bleed = Math.max(minBleed, left + 8);
  const d = `M ${-bleed} ${y0} H ${x - radius} Q ${x} ${y0} ${x} ${y0 + radius} V ${end}`;
  const stations = stops.map((s) => (s.top + s.bottom) / 2);
  const spurs = stops.map((s, i): [number, number] => [s.right, stations[i]!]);
  return { width, height: Math.max(height, end + 20), d, x, stations, spurs, end };
}

export function RailLine({ container, stops }: Props) {
  const [geo, setGeo] = useState<Geometry | null>(null);

  // A passive effect, not a layout effect: React attaches the parent's ref (the container) only
  // after its children's layout effects have run.
  useEffect(() => {
    const root = container.current;
    if (!root) return;
    const update = () => {
      const box = root.getBoundingClientRect();
      const rects = stops
        .map((s) => s.current?.getBoundingClientRect())
        .filter((r): r is DOMRect => !!r)
        .map((r) => ({ top: r.top - box.top, bottom: r.bottom - box.top, right: r.right - box.left }));
      setGeo(measure(box.width, box.height, rects, box.left));
    };
    update();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(update);
    ro.observe(root);
    stops.forEach((s) => s.current && ro.observe(s.current));
    document.fonts?.ready.then(update).catch(() => {});
    return () => ro.disconnect();
  }, [container, stops]);

  if (!geo) return null;
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      className="pointer-events-none absolute top-0 left-0 overflow-visible"
      width={geo.width}
      height={geo.height}
      data-testid="rail-line"
    >
      <path d={geo.d} fill="none" strokeWidth={9} strokeLinecap="round" className="stroke-line-soft" />
      <path d={geo.d} fill="none" strokeWidth={9} strokeLinecap="round" pathLength={1} className="rail-track stroke-line" />
      {geo.spurs.map(([x1, y], i) => (
        <line key={`spur-${i}`} x1={x1} y1={y} x2={geo.x} y2={y} strokeWidth={4} className="stroke-line-soft" />
      ))}
      {geo.stations.map((y, i) => (
        <circle key={`station-${i}`} cx={geo.x} cy={y} r={10} strokeWidth={5} className="fill-paper stroke-line" />
      ))}
      <circle cx={geo.x} cy={geo.end} r={14} strokeWidth={5} className="fill-paper stroke-line" />
      <g className="rail-train" style={{ offsetPath: `path('${geo.d}')` }}>
        <circle r={14} strokeWidth={5} className="fill-sun stroke-paper" />
      </g>
    </svg>
  );
}
