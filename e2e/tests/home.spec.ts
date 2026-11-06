import { expect, test } from './fixtures';
import { Site } from '../pages/site';

test('the start screen shows both ways in on the rail line', async ({ page }) => {
  const site = new Site(page);
  await site.open('/');
  await expect(site.heading()).toHaveText('Vind de trein waarin je zit.');
  await expect(page).toHaveTitle('Vind de trein waarin je zit · dezetrein');
  await expect(page.getByRole('button', { name: /Vind mijn trein/ })).toBeVisible();
  await expect(page.getByRole('button', { name: /Treinstelnummer/ })).toBeVisible();
  await expect(page.getByTestId('rail-line')).toBeAttached();
  await expect(page.getByText('dezetrein is niet verbonden aan NS')).toBeVisible();
});

test('language and theme are remembered', async ({ page }) => {
  const site = new Site(page);
  await site.open('/');
  await site.switchLanguage();
  await expect(site.heading()).toHaveText('Find the train you are on.');
  await expect(page.locator('html')).toHaveAttribute('lang', 'en');
  await site.cycleTheme(); // system to light
  await site.cycleTheme(); // light to dark
  await expect(page.locator('html')).toHaveClass(/dark/);
  await page.reload();
  await expect(site.heading()).toHaveText('Find the train you are on.');
  await expect(page.locator('html')).toHaveClass(/dark/);
});

test('every request stays on the site itself: no third parties, fonts included', async ({ page, baseURL }) => {
  const origins = new Set<string>();
  page.on('request', (r) => origins.add(new URL(r.url()).origin));
  await new Site(page).open('/');
  await page.waitForLoadState('networkidle');
  expect([...origins]).toEqual([new URL(baseURL!).origin]);
  const fonts = await page.evaluate(() => [...document.fonts].filter((f) => f.status === 'loaded').map((f) => f.family));
  expect(fonts.join(' ')).toMatch(/Bricolage Grotesque/);
});

test('the site and the API send their security headers', async ({ request }) => {
  const site = await request.get('/');
  expect(site.headers()['content-security-policy']).toContain("default-src 'self'");
  expect(site.headers()['permissions-policy']).toContain('geolocation=(self)');
  const api = await request.get('/api/does-not-exist');
  expect(api.status()).toBe(404);
  expect(await api.json()).toEqual({ error: 'not_found' });
  expect(api.headers()['content-security-policy']).toBe("default-src 'none'; frame-ancestors 'none'");
});

test('an unknown page says so and leads back', async ({ page }) => {
  const site = new Site(page);
  await site.open('/bestaat-niet');
  await expect(site.heading()).toHaveText('Pagina niet gevonden');
  await page.getByRole('link', { name: /Naar de start/ }).click();
  await expect(site.heading()).toHaveText('Vind de trein waarin je zit.');
});
