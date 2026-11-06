import { test as base, expect } from '@playwright/test';
import { randomBytes } from 'node:crypto';

// Every test is its own client to the backend's rate limiter: Cloudflare would set
// CF-Connecting-IP, and nginx passes it on, so a random documentation-range address per test
// keeps parallel tests from sharing (and exhausting) one bucket.
export const test = base.extend({
  context: async ({ context }, use) => {
    const ip = `198.51.100.${1 + (randomBytes(1)[0]! % 254)}`;
    await context.setExtraHTTPHeaders({ 'CF-Connecting-IP': ip });
    await use(context);
  },
});

export { expect };

// Fixture data (backend/testdata/ns, recorded 2026-11-04): IC 2450 Lelystad to Rotterdam, with
// trainsets 3304 and 3108, passing Schiphol with a platform change from 5-6 to 6.
export const ic2450 = {
  train: '2450',
  destination: 'Rotterdam Centraal',
  carriages: ['3304', '3108'],
  // Its recorded position, as the phone reports it inside the train (coarse).
  position: { latitude: 52.0779, longitude: 4.5652, accuracy: 1414 },
  // A moment on that trip: between Amsterdam Zuid and Schiphol.
  between: new Date('2026-11-04T14:52:00Z'),
};

export const utrecht = { latitude: 52.0894, longitude: 5.1101, accuracy: 30 };
