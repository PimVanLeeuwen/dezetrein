import { expect, test, utrecht } from './fixtures';
import { allowLocation, OtherPage } from '../pages/site';

test('opens a train by its journey number', async ({ page }) => {
  await new OtherPage(page).journey('2450');
  await expect(page).toHaveURL(/\/trein\/2450$/);
});

test('lists the stations nearby, their departures, and opens a train', async ({ page, context }) => {
  await allowLocation(context, utrecht);
  const other = new OtherPage(page);
  await other.open('/zoek');
  await page.getByRole('button', { name: 'Zoek stations in de buurt' }).click();
  const stations = page.getByRole('listitem').getByRole('button');
  await expect(stations.first()).toContainText('Utrecht Centraal');
  await stations.first().click();
  await expect(page.getByRole('heading', { level: 2, name: 'Vertrek Utrecht Centraal' })).toBeVisible();
  const departure = page.getByRole('button', { name: / naar / }).first();
  await expect(departure).toBeVisible();
  await departure.click();
  await expect(page).toHaveURL(/\/trein\/\d+$/);
});
