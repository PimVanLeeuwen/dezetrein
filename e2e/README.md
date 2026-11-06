# End-to-end tests

Playwright drives the built site (nginx, with its real Content-Security-Policy) against the Go backend in fixture mode, both started from [`docker-compose.e2e.yml`](../docker-compose.e2e.yml). The backend replays the NS recordings in [`backend/testdata/ns`](../backend/testdata/ns) (recorded 2026-11-04), so the suite needs no NS key and sees the same data on every run.

Every test runs in three browsers: desktop Chromium, Android Chromium (Pixel 7) and iPhone WebKit (iPhone 15), with locale `nl-NL` and the `Europe/Amsterdam` time zone.

## Running

```bash
npm ci && npx playwright install chromium webkit
```

```bash
npm test
```

`npm test` builds and starts the stack, waits for it and stops it afterwards. To keep it running between runs, start it once with `npm run stack:up`, then run with `E2E_NO_WEBSERVER=1 npm test`, and stop it with `npm run stack:down`. Rebuild with `npm run stack:up` after changing the site or the backend: a running stack keeps the old images.

## How the tests stay independent

- **Location** comes from Playwright's `geolocation` option and permission grants (`allowLocation` in [`pages/site.ts`](pages/site.ts)); a test without a grant gets the browser's denial.
- **Rate limits.** Each test sends its own `CF-Connecting-IP` (a documentation-range address, see [`tests/fixtures.ts`](tests/fixtures.ts)), so parallel tests never share a rate-limit bucket.
- **Time.** The trip tests set the browser clock to 2026-11-04 15:52 Dutch time, when IC 2450 is between Amsterdam Zuid and Schiphol in the recording. The find tests use the real clock, as the backend checks that readings are recent.
- **Shared server state.** The backend remembers trainsets it has seen for 10 minutes. The trainset test accepts both outcomes (found at once, or found after sharing the location), since another test may have viewed the same train.

## Coverage map

| Area | Spec | What is checked |
|------|------|-----------------|
| Start screen | [home](tests/home.spec.ts) | Both ways in, the rail line, the NS note; language and theme remembered after a reload |
| Privacy | [home](tests/home.spec.ts) | Every request during a page load goes to the site itself (no third parties); self-hosted fonts load |
| Security headers | [home](tests/home.spec.ts) | The site's CSP and Permissions-Policy; the API's own strict CSP, once; JSON 404 |
| Not found | [home](tests/home.spec.ts) | Unknown page, and the way back |
| Find my train | [find](tests/find.spec.ts) | Coarse location (±1414 m) to "Zit je in de trein naar Rotterdam Centraal?" with its trainsets, then the trip; the location only in a POST body |
| Find: alternatives | [find](tests/find.spec.ts) | Picking another train from "Of zit je in een van deze?"; "Nee, zoek opnieuw" starts over |
| Find: permissions | [find](tests/find.spec.ts) | Allowed already: the explanation is skipped; denied: explained, with links to the other ways; outside the network |
| Trip view | [trip](tests/trip.spec.ts) | Delay, "Je bent hier", next stop, platform change 5-6 to 6, trainsets, disruption on the route, stations behind |
| Trip: your trainset | [trip](tests/trip.spec.ts) | `?rijtuig=` marks the trainset |
| Trip: refresh | [trip](tests/trip.spec.ts) | A new request after 30 s |
| Trip: unknown | [trip](tests/trip.spec.ts) | A journey number that does not run |
| Trainset search | [carriage](tests/carriage.spec.ts) | 3304 to IC 2450 with the trainset marked; an unknown trainset after sharing the location; digits only |
| Other ways | [other](tests/other.spec.ts) | Journey number to the trip; nearest stations to Utrecht Centraal departures to a train |
| Accessibility | [a11y](tests/a11y.spec.ts) | axe-core (WCAG 2.2 A/AA, colour contrast as rendered) on every page, the trip view and the confirmation question, in the light and dark theme |

## Not covered here

- The live NS API: the backend's own tests cover the client against recordings, and the field test covers real trips.
- Screen readers and keyboard flow on a real phone: automated checks cannot prove those; they are part of the field test before launch.
