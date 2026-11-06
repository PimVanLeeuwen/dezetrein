import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';

// Page objects: what a traveller sees and taps, in Dutch (the suite runs with locale nl-NL).

export class Site {
  constructor(readonly page: Page) {}

  heading(): Locator {
    return this.page.getByRole('heading', { level: 1 });
  }

  async open(path = '/') {
    await this.page.goto(path);
    await expect(this.heading()).toBeVisible();
  }

  async switchLanguage() {
    await this.page.getByRole('button', { name: /Switch to English|Wissel naar Nederlands/ }).click();
  }

  async cycleTheme() {
    await this.page.getByRole('button', { name: /^(Thema|Theme):/ }).click();
  }
}

// allowLocation gives the page a location, as a phone that shares it would.
export async function allowLocation(context: BrowserContext, at: { latitude: number; longitude: number; accuracy: number }) {
  await context.grantPermissions(['geolocation']);
  await context.setGeolocation(at);
}

export class FindPage extends Site {
  // start taps "Vind mijn trein". With location already allowed the flow starts by itself;
  // otherwise it explains first and waits for share().
  async start() {
    await this.open('/');
    await this.page.getByRole('button', { name: /Vind mijn trein/ }).click();
    await expect(this.page).toHaveURL(/\/vind$/);
  }

  async share() {
    await this.page.getByRole('button', { name: /Deel mijn locatie/ }).click();
  }

  // The answer comes after a few seconds (early when clear, at 15 s otherwise).
  async question(): Promise<Locator> {
    const h = this.heading();
    await expect(h).toHaveText(/^Zit je in/, { timeout: 25_000 });
    return h;
  }
}

export class TripPage extends Site {
  nextStop(): Locator {
    return this.page.getByText('Volgende station').locator('..');
  }

  trainsets(): Locator {
    return this.page.getByRole('list', { name: /^Treinstellen:/ });
  }
}

export class CarriagePage extends Site {
  async search(n: string) {
    await this.open('/treinstel');
    await this.page.getByLabel('Treinstelnummer').fill(n);
    await this.page.getByRole('button', { name: /Zoek deze trein/ }).click();
  }
}

export class OtherPage extends Site {
  async journey(n: string) {
    await this.open('/zoek');
    await this.page.getByLabel('Ritnummer').fill(n);
    await this.page.getByRole('button', { name: /Bekijk deze trein/ }).click();
  }
}
