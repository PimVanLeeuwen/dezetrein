import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import type { Departures, NearestStations } from '../api/types';
import { I18nProvider } from '../i18n/i18n';
import { fakeGeo } from '../test/fakes';
import { Other } from './Other';

beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl']));
afterEach(() => {
  window.history.replaceState(null, '', '/');
  vi.restoreAllMocks();
});

const stations: NearestStations = {
  stations: [
    { code: 'UT', name: 'Utrecht Centraal', distanceM: 40 },
    { code: 'UTVR', name: 'Utrecht Vaartsche Rijn', distanceM: 1640 },
  ],
  as_of: '2026-11-06T12:00:00Z',
  stale: false,
};

const departures: Departures = {
  station: 'UT',
  departures: [
    { train: '3539', category: 'IC', direction: 'Venlo', planned: '2026-11-06T13:51:00+01:00', actual: '2026-11-06T13:54:00+01:00', plannedTrack: '18', actualTrack: '19', cancelled: false },
    { train: '5639', category: 'SPR', direction: 'Zwolle', planned: '2026-11-06T13:52:00+01:00', plannedTrack: '4', actualTrack: '4', cancelled: false },
  ],
  as_of: '2026-11-06T12:50:00Z',
  stale: false,
};

function setup(over: Partial<{ nearestStations: () => Promise<NearestStations>; departures: () => Promise<Departures> }> = {}) {
  const g = fakeGeo();
  const client = { nearestStations: vi.fn(over.nearestStations ?? (async () => stations)), departures: vi.fn(over.departures ?? (async () => departures)) };
  render(
    <I18nProvider>
      <Other client={client} geo={g.geo} />
    </I18nProvider>,
  );
  return { g, client };
}

describe('Other ways', () => {
  it('opens a train by journey number', () => {
    setup();
    fireEvent.change(screen.getByLabelText('Ritnummer'), { target: { value: '3522' } });
    fireEvent.click(screen.getByRole('button', { name: /Bekijk deze trein/ }));
    expect(window.location.pathname).toBe('/trein/3522');
  });

  it('rejects a journey number that is not digits', () => {
    setup();
    fireEvent.change(screen.getByLabelText('Ritnummer'), { target: { value: 'IC' } });
    fireEvent.click(screen.getByRole('button', { name: /Bekijk deze trein/ }));
    expect(screen.getByText('Typ alleen cijfers, maximaal 6.')).toBeInTheDocument();
    expect(window.location.pathname).toBe('/');
  });

  it('finds stations nearby, lists departures, and opens a train', async () => {
    const { g, client } = setup();
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    expect(screen.getByRole('status')).toHaveTextContent('Stations in de buurt zoeken');
    await act(async () => g.fix());
    expect(client.nearestStations).toHaveBeenCalledWith([expect.objectContaining({ lat: 52.09 })], expect.any(AbortSignal));
    expect(screen.getByRole('button', { name: /Utrecht Vaartsche Rijn/ })).toHaveTextContent('op 1,6 km');
    await act(async () => fireEvent.click(screen.getByRole('button', { name: /Utrecht Centraal/ })));
    expect(screen.getByRole('heading', { level: 2, name: 'Vertrek Utrecht Centraal' })).toBeInTheDocument();
    const venlo = screen.getByRole('button', { name: '13:54 IC naar Venlo, spoor 19' });
    expect(venlo).toHaveTextContent('13:51');
    fireEvent.click(screen.getByRole('button', { name: 'Ander station' }));
    expect(screen.getByRole('button', { name: /Utrecht Vaartsche Rijn/ })).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: /Utrecht Centraal/ })));
    fireEvent.click(screen.getByRole('button', { name: /Zwolle/ }));
    expect(window.location.pathname).toBe('/trein/5639');
  });

  it('goes back to the stations when departures fail to load', async () => {
    const { g } = setup({ departures: async () => Promise.reject(new Error('down')) });
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    await act(async () => g.fix());
    await act(async () => fireEvent.click(screen.getByRole('button', { name: /Utrecht Centraal/ })));
    expect(screen.getByRole('button', { name: /Utrecht Vaartsche Rijn/ })).toBeInTheDocument();
  });

  it('says when there is no station nearby', async () => {
    const { g } = setup({ nearestStations: async () => ({ ...stations, stations: [] }) });
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    await act(async () => g.fix());
    expect(screen.getByText(/Geen station met vertrektijden/)).toBeInTheDocument();
  });

  it('explains a denied location, and lets the user try again', async () => {
    const { g } = setup({ nearestStations: async () => Promise.reject(new ApiError(503, 'unavailable')) });
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    await act(async () => g.fail('denied'));
    expect(screen.getByText(/geeft je locatie niet door/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Zoek stations in de buurt' }));
    await act(async () => g.fix());
    expect(screen.getByText(/even niet beschikbaar/)).toBeInTheDocument();
  });
});
