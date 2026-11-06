import { expect, ic2450, test } from './fixtures';
import { TripPage } from '../pages/site';

// The fixtures were recorded on 2026-11-04; the browser clock is set to that afternoon so IC
// 2450 is on its way, between Amsterdam Zuid and Schiphol.
test.beforeEach(async ({ page }) => {
  await page.clock.install({ time: ic2450.between });
});

test('shows where the train is, the next stop with its delay and platform change, and its trainsets', async ({ page }) => {
  const trip = new TripPage(page);
  await trip.open('/trein/2450');
  await expect(trip.heading()).toContainText('Intercity direct 2450');
  await expect(trip.heading()).toContainText(`naar ${ic2450.destination}`);
  await expect(page.getByLabel('16 minuten vertraging')).toBeVisible();
  await expect(page.getByText('Je bent hier')).toBeVisible();
  await expect(trip.nextStop()).toContainText('Schiphol Airport');
  await expect(page.getByText('Ander spoor: was 5-6')).toBeVisible();
  await expect(trip.trainsets()).toHaveAccessibleName(`Treinstellen: ${ic2450.carriages.join(', ')}`);
  await expect(page.getByRole('region', { name: /^Storing:/ }).first()).toBeVisible();
  await page.getByRole('button', { name: /stations geweest/ }).click();
  await expect(page.getByText('Lelystad Centrum')).toBeVisible();
});

test('marks the trainset the user searched by', async ({ page }) => {
  const trip = new TripPage(page);
  await trip.open('/trein/2450?rijtuig=3108');
  const mine = trip.trainsets().getByRole('listitem').filter({ hasText: '3108' });
  await expect(mine).toHaveAttribute('aria-current', 'true');
  await expect(mine).toContainText('jouw treinstel');
});

test('refreshes every 30 seconds while the page is open', async ({ page }) => {
  const trip = new TripPage(page);
  let loads = 0;
  page.on('request', (r) => {
    if (r.url().includes('/api/trains/2450')) loads++;
  });
  await trip.open('/trein/2450');
  await expect(trip.nextStop()).toBeVisible();
  const before = loads;
  await page.clock.runFor(30_000);
  await expect.poll(() => loads).toBeGreaterThan(before);
});

test('explains an unknown journey number', async ({ page }) => {
  const trip = new TripPage(page);
  await trip.open('/trein/99999');
  await expect(trip.heading()).toHaveText('Trein 99999 niet gevonden', { timeout: 15_000 });
  await expect(page.getByText(/niet het nummer van het treinstel/)).toBeVisible();
});
