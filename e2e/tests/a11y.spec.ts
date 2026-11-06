import AxeBuilder from '@axe-core/playwright';
import { expect, ic2450, test } from './fixtures';
import { allowLocation, FindPage } from '../pages/site';

// axe-core in a real browser, so colour contrast is checked as rendered, in both themes. Reduced
// motion keeps the fade-in from being measured halfway.
test.use({ reducedMotion: 'reduce' });

const tags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22a', 'wcag22aa'];

async function check(page: import('@playwright/test').Page) {
  const result = await new AxeBuilder({ page }).withTags(tags).analyze();
  expect(result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)).toEqual([]);
}

for (const scheme of ['light', 'dark'] as const) {
  test.describe(`${scheme} theme`, () => {
    test.use({ colorScheme: scheme });

    for (const path of ['/', '/vind', '/treinstel', '/zoek', '/privacy', '/over', '/bestaat-niet']) {
      test(`page ${path}`, async ({ page }) => {
        await page.goto(path);
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
        await check(page);
      });
    }

    test('trip view', async ({ page }) => {
      await page.clock.install({ time: ic2450.between });
      await page.goto('/trein/2450?rijtuig=3304');
      await expect(page.getByText('Volgende station')).toBeVisible();
      await check(page);
    });

    test('the confirmation question', async ({ page, context }) => {
      await allowLocation(context, ic2450.position);
      const find = new FindPage(page);
      await find.start();
      await find.question();
      await check(page);
    });
  });
}
