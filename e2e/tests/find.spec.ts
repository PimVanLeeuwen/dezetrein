import { expect, ic2450, test } from './fixtures';
import { allowLocation, FindPage } from '../pages/site';

test('finds the train from a coarse location, confirms it and opens the trip', async ({ page, context }) => {
  await allowLocation(context, ic2450.position);
  const find = new FindPage(page);
  await find.start();
  const matchRequest = page.waitForRequest((r) => r.url().includes('/api/match'));
  // Location is allowed already, so the search starts without the explanation.
  const req = await matchRequest;
  // The location goes in the body of a POST, never in the address.
  expect(req.method()).toBe('POST');
  expect(req.url()).not.toContain(String(ic2450.position.latitude));
  expect(req.postDataJSON().fixes[0]).toMatchObject({ lat: ic2450.position.latitude, lng: ic2450.position.longitude });

  await expect(await find.question()).toHaveText(`Zit je in de trein naar ${ic2450.destination}?`);
  await expect(page.getByLabel(`Treinstellen: ${ic2450.carriages.join(', ')}`)).toBeVisible();
  await page.getByRole('button', { name: /Ja, deze trein/ }).click();
  await expect(page).toHaveURL(/\/trein\/2450$/);
  await expect(find.heading()).toContainText(`naar ${ic2450.destination}`);
});

test('offers other trains nearby, and opens the one picked', async ({ page, context }) => {
  await allowLocation(context, ic2450.position);
  const find = new FindPage(page);
  await find.start();
  await find.question();
  const others = page.getByRole('region').filter({ has: page.getByRole('heading', { name: 'Of zit je in een van deze?' }) });
  const first = others.getByRole('button').first();
  await expect(first).toBeVisible();
  const label = await first.textContent();
  const train = /\d+/.exec(label ?? '')?.[0];
  await first.click();
  await expect(page).toHaveURL(new RegExp(`/trein/${train}$`));
});

test('searches again on no', async ({ page, context }) => {
  await allowLocation(context, ic2450.position);
  const find = new FindPage(page);
  await find.start();
  await find.question();
  await page.getByRole('button', { name: 'Nee, zoek opnieuw' }).click();
  await expect(find.heading()).toHaveText('We zoeken je trein');
});

test('skips the explanation when location is already allowed', async ({ page, context }) => {
  await allowLocation(context, ic2450.position);
  const find = new FindPage(page);
  await find.open('/vind');
  await expect(find.heading()).toHaveText('We zoeken je trein');
});

test('explains first, then a denied location, and offers the other ways', async ({ page, context }) => {
  await context.clearPermissions();
  const find = new FindPage(page);
  await find.start();
  await expect(find.heading()).toHaveText('Waar ben je?');
  await find.share();
  await expect(find.heading()).toHaveText('Je locatie staat uit', { timeout: 25_000 });
  await expect(page.getByRole('link', { name: 'Zoek op treinstelnummer' })).toHaveAttribute('href', '/treinstel');
  await expect(page.getByRole('link', { name: 'Andere manieren' })).toHaveAttribute('href', '/zoek');
});

test('says so outside the rail network', async ({ page, context }) => {
  await allowLocation(context, { latitude: 40.4168, longitude: -3.7038, accuracy: 20 });
  const find = new FindPage(page);
  await find.start();
  await expect(find.heading()).toHaveText('Je bent buiten het spoornet');
});
