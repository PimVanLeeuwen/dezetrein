import { describe, expect, it } from 'vitest';
import css from '../index.css?raw';

// Checks the palette in index.css against WCAG 2.2 AA, in both themes, for every pairing the
// components use: 4.5:1 for text, 3:1 for large text and for UI parts (lines, focus rings).

function palette(selector: string): Record<string, string> {
  const block = css.match(new RegExp(`(?:^|\\n)${selector.replace('.', '\\.')} \\{([^}]*)\\}`))?.[1] ?? '';
  return Object.fromEntries([...block.matchAll(/--([\w-]+):\s*(#[0-9a-f]{6})/gi)].map((m) => [m[1]!, m[2]!]));
}

function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!;
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi! + 0.05) / (lo! + 0.05);
}

const pairs: [fg: string, bg: string, min: number][] = [
  ['ink', 'paper', 4.5],
  ['ink', 'card', 4.5],
  ['dim', 'paper', 4.5],
  ['dim', 'card', 4.5],
  ['line', 'paper', 4.5], // accent text, the "soon" label
  ['line', 'card', 3], // borders and the focus ring on cards
  ['on-line', 'line', 4.5], // primary button text
  ['late', 'paper', 4.5],
  ['late', 'card', 4.5],
  ['on-late', 'late', 4.5],
];

describe.each([
  ['light', ':root'],
  ['dark', '.dark'],
])('%s palette', (_, selector) => {
  const p = palette(selector);
  it('is complete', () => {
    for (const [fg, bg] of pairs) expect(p[fg] && p[bg], `${fg} on ${bg}`).toBeTruthy();
  });
  it.each(pairs)('%s on %s meets %f:1', (fg, bg, min) => {
    expect(contrast(p[fg]!, p[bg]!)).toBeGreaterThanOrEqual(min);
  });
});
