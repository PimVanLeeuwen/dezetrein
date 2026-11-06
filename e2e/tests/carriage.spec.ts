import { expect, ic2450, test, utrecht } from './fixtures';
import { allowLocation, CarriagePage } from '../pages/site';

test('finds the train by trainset number and marks it in the trip', async ({ page, context }) => {
  await allowLocation(context, ic2450.position);
  const search = new CarriagePage(page);
  await search.search('3304');
  // The backend may know the trainset already (another test viewed this train) or not; without
  // a recent sighting it asks for the location first.
  const ask = page.getByRole('button', { name: /Gebruik mijn locatie/ });
  await expect(ask.or(page.getByText('jouw treinstel'))).toBeVisible({ timeout: 15_000 });
  if (await ask.isVisible()) await ask.click();
  await expect(page).toHaveURL(/\/trein\/2450\?rijtuig=3304$/, { timeout: 15_000 });
  await expect(page.getByRole('listitem').filter({ hasText: '3304' })).toHaveAttribute('aria-current', 'true');
});

test('asks for a location when the trainset is unknown, then says none nearby has it', async ({ page, context }) => {
  await allowLocation(context, utrecht);
  const search = new CarriagePage(page);
  await search.search('9999');
  await expect(search.heading()).toHaveText('Waar ben je ongeveer?');
  await page.getByRole('button', { name: /Gebruik mijn locatie/ }).click();
  await expect(search.heading()).toHaveText('Treinstel 9999 niet gevonden', { timeout: 15_000 });
  await expect(page.getByRole('link', { name: 'Andere manieren' })).toBeVisible();
});

test('only takes digits', async ({ page }) => {
  const search = new CarriagePage(page);
  await search.search('IC');
  await expect(page.getByText('Typ alleen cijfers, maximaal 6.')).toBeVisible();
});
