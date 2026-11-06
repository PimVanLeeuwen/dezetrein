import axe from 'axe-core';

// expectAccessible runs axe-core on a rendered page and fails with a readable list of every
// WCAG 2.2 A/AA violation. Colour contrast is left out: jsdom has no rendering, so axe cannot
// measure it; src/lib/contrast.test.ts checks the palette in both themes instead.
export async function expectAccessible(node: Element = document.body) {
  const result = await axe.run(node, {
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22a', 'wcag22aa'] },
    rules: { 'color-contrast': { enabled: false } },
  });
  if (result.violations.length === 0) return;
  const lines = result.violations.flatMap((v) => [`${v.id} (${v.impact}): ${v.help}`, ...v.nodes.slice(0, 3).map((n) => `  at ${n.target.join(' ')}`)]);
  throw new Error(`accessibility violations:\n${lines.join('\n')}`);
}
