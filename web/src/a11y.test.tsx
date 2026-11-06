// Automated accessibility checks (axe-core, WCAG 2.2 A/AA) on every page and state, inside the
// real layout so landmarks, headings and the header controls are checked too.
import { act, fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, it, vi } from 'vitest';
import type { NearestStations } from './api/types';
import { Layout } from './components/Layout';
import { about, privacy } from './content/pages';
import { ContentPage } from './components/Prose';
import type { FindState } from './flow/useFindTrain';
import type { TripState } from './flow/useTrip';
import { I18nProvider } from './i18n/i18n';
import { CarriageView, type CarriageState } from './pages/Carriage';
import { FindView } from './pages/Find';
import { Home } from './pages/Home';
import { Other } from './pages/Other';
import { NotFound } from './pages/Simple';
import { TripView } from './pages/Trip';
import { expectAccessible } from './test/axe';
import { candidate, fakeGeo, result } from './test/fakes';
import { at, tripFixture } from './test/trip';

let lang = 'nl';
beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockImplementation(() => [lang]));
afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

const page = (ui: ReactNode) =>
  render(
    <I18nProvider>
      <Layout>{ui}</Layout>
    </I18nProvider>,
  );

const noop = () => {};
const find = (state: FindState) => <FindView state={state} start={noop} reset={noop} />;
const trip = (state: TripState, mine?: string) => <TripView train="2450" state={state} now={at.betweenZuidAndSchiphol} reload={noop} mine={mine} />;
const carriage = (state: CarriageState) => <CarriageView state={state} search={async () => {}} searchNearby={async () => {}} reset={noop} />;
const ready: TripState = { status: 'ready', trip: tripFixture, refreshing: false, refreshFailed: false };

const pages: [string, ReactNode][] = [
  ['home', <Home />],
  ['find: intro', find({ phase: 'intro' })],
  ['find: locating', find({ phase: 'locating', readings: 3, accuracyM: 1414, checking: true })],
  ['find: confirm', find({ phase: 'confirm', result: result('low', [candidate('3539', 'Venlo', ['9574']), candidate('5639', 'Zwolle', ['2416']), candidate('8940')]) })],
  ['find: denied', find({ phase: 'failed', reason: 'denied' })],
  ['find: api error', find({ phase: 'failed', reason: 'api', message: 'error.network' })],
  ['trip: loading', trip({ status: 'loading' })],
  ['trip: ready', trip(ready, '3108')],
  ['trip: stale', trip({ ...ready, trip: { ...tripFixture, stale: true, disruptions: [...tripFixture.disruptions, { id: 'm', type: 'MAINTENANCE', title: 'Utrecht - Leiden.' }] }, refreshFailed: true })],
  ['trip: not found', trip({ status: 'notFound' })],
  ['trip: failed', trip({ status: 'failed', message: 'error.unavailable' })],
  ['carriage: form', carriage({ phase: 'form' })],
  ['carriage: searching', carriage({ phase: 'searching', carriage: '9402' })],
  ['carriage: need location', carriage({ phase: 'needLocation', carriage: '9402' })],
  ['carriage: not found', carriage({ phase: 'notFound', carriage: '9402', searched: 7 })],
  ['privacy', <ContentPage content={privacy.nl} />],
  ['about', <ContentPage content={about.nl} />],
  ['not found', <NotFound />],
];

describe('accessibility (axe, WCAG 2.2 A/AA)', () => {
  it.each(pages)('%s', async (_, ui) => {
    page(ui);
    await expectAccessible();
  });

  it('the English start screen', async () => {
    lang = 'en';
    page(<Home />);
    await expectAccessible();
    lang = 'nl';
  });

  it('other ways, through stations to departures', async () => {
    const g = fakeGeo();
    const stations: NearestStations = { stations: [{ code: 'UT', name: 'Utrecht Centraal', distanceM: 40 }, { code: 'UTZL', name: 'Utrecht Zuilen', distanceM: 2047 }], as_of: '', stale: false };
    const client = {
      nearestStations: async () => stations,
      departures: async () => ({ station: 'UT', departures: [{ train: '3539', category: 'IC', direction: 'Venlo', planned: '2026-11-06T13:51:00+01:00', plannedTrack: '18', cancelled: false }], as_of: '', stale: false }),
    };
    page(<Other client={client} geo={g.geo} />);
    await expectAccessible();
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    await act(async () => g.fix());
    await expectAccessible();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: /Utrecht Centraal/ })));
    await expectAccessible();
  });
});
