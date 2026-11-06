import { fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { TripState } from '../flow/useTrip';
import { I18nProvider } from '../i18n/i18n';
import { at, tripFixture } from '../test/trip';
import { TripView } from './Trip';

beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl']));
afterEach(() => vi.restoreAllMocks());

function view(state: TripState, now = at.betweenZuidAndSchiphol, mine?: string) {
  const reload = vi.fn();
  render(
    <I18nProvider>
      <TripView train="2450" state={state} now={now} reload={reload} mine={mine} />
    </I18nProvider>,
  );
  return reload;
}

const ready = (over: Partial<TripState & { status: 'ready' }> = {}): TripState => ({ status: 'ready', trip: tripFixture, refreshing: false, refreshFailed: false, ...over });

describe('TripView', () => {
  it('shows the train, where it goes and how late it is', () => {
    view(ready());
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toHaveTextContent('Intercity direct 2450');
    expect(h1).toHaveTextContent('naar Rotterdam Centraal');
    expect(document.title).toBe('Trein 2450 naar Rotterdam Centraal · dezetrein');
    expect(screen.getByLabelText('16 minuten vertraging')).toHaveTextContent('+16 min');
    expect(screen.getByText('ICNG')).toBeInTheDocument();
  });

  it('shows the next stop with expected time, planned time and the platform change', () => {
    view(ready());
    expect(screen.getByText('Je bent hier')).toBeInTheDocument();
    expect(screen.getByText('Volgende station')).toBeInTheDocument();
    expect(screen.getByText('Schiphol Airport')).toBeInTheDocument();
    expect(screen.getByText('15:57')).toBeInTheDocument();
    expect(screen.getByLabelText('gepland 15:42')).toHaveTextContent('15:42');
    expect(screen.getByText('Ander spoor: was 5-6')).toBeInTheDocument();
  });

  it('lists the stops ahead, and folds away the ones behind', () => {
    view(ready());
    expect(screen.getAllByText('Rotterdam Centraal').map((e) => e.tagName)).toEqual(['H1', 'P']); // heading and stop row
    expect(screen.getByText('16:23')).toBeInTheDocument();
    expect(screen.queryByText('Almere Buiten')).toBeNull();
    const toggle = screen.getByRole('button', { name: '3 stations geweest' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    fireEvent.click(toggle);
    expect(screen.getByText('Almere Buiten')).toBeInTheDocument();
    expect(screen.queryByText('Almere Oostvaarders')).toBeNull(); // passing station
  });

  it('shows carriages, crowding and disruptions', () => {
    view(ready());
    const stock = screen.getByRole('list', { name: 'Treinstellen: 3304, 3108' });
    expect(within(stock).getAllByRole('listitem')).toHaveLength(2);
    expect(screen.getByText(/2 treinstellen · 161 m lang/)).toBeInTheDocument();
    expect(screen.getAllByText('rustig').length).toBeGreaterThan(0);
    expect(screen.getByRole('region', { name: 'Storing: Rotterdam - Breda.' })).toHaveTextContent('wisselstoring');
  });

  it('folds engineering works and general notices below the trip', () => {
    const notices = [
      { id: 'm1', type: 'MAINTENANCE', title: 'Utrecht - Leiden.' },
      { id: 'c1', type: 'CALAMITY', title: 'Werkzaamheden Rotterdam Centraal', url: 'https://www.ns.nl/x' },
      { id: 'x1', type: 'SOMETHING_NEW', title: 'Iets nieuws' },
    ];
    view(ready({ trip: { ...tripFixture, disruptions: notices } }));
    expect(screen.queryByRole('region', { name: /Storing/ })).toBeNull();
    const summary = screen.getByText('Werkzaamheden en meldingen (3)');
    expect(summary.closest('details')).not.toHaveAttribute('open');
    expect(screen.getByRole('region', { name: 'Werkzaamheden: Utrecht - Leiden.', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Let op: Iets nieuws', hidden: true })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Meer bij NS', hidden: true })).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('marks the trainset the user searched by', () => {
    view(ready(), at.betweenZuidAndSchiphol, '3108');
    const mine = screen.getByText('3108').closest('li')!;
    expect(mine).toHaveAttribute('aria-current', 'true');
    expect(mine).toHaveTextContent('jouw treinstel');
    expect(screen.getByText('3304').closest('li')).not.toHaveAttribute('aria-current');
  });

  it('says how old the data is, and when it may be stale', () => {
    view(ready({ trip: { ...tripFixture, stale: true }, refreshFailed: true }));
    expect(screen.getByText('NS-gegevens van 15:52, ververst elke 30 seconden')).toBeInTheDocument();
    expect(screen.getByText(/Mogelijk verouderd/)).toBeInTheDocument();
    expect(screen.getByText(/Verversen lukte niet/)).toBeInTheDocument();
  });

  it('before departure and after arrival', () => {
    view(ready(), at.before);
    expect(screen.getByText('Vertrekt nog')).toBeInTheDocument();
    expect(screen.getByText('Vertrek')).toBeInTheDocument();
    expect(screen.queryByText('Je bent hier')).toBeNull();
  });

  it('after arrival', () => {
    view(ready(), at.arrived);
    expect(screen.getByText('Aangekomen')).toBeInTheDocument();
    expect(screen.getByText('Eindstation')).toBeInTheDocument();
  });

  it('loading, not found and failed', () => {
    view({ status: 'loading' });
    expect(screen.getByRole('status', { name: 'Reisinformatie laden…' })).toBeInTheDocument();
  });

  it('explains an unknown train number', () => {
    view({ status: 'notFound' });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Trein 2450 niet gevonden');
    expect(screen.getByText(/niet het nummer van het treinstel/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Vind mijn trein/ })).toHaveAttribute('href', '/vind');
  });

  it('offers a retry after an error', () => {
    const reload = view({ status: 'failed', message: 'error.unavailable' });
    fireEvent.click(screen.getByRole('button', { name: 'Probeer opnieuw' }));
    expect(reload).toHaveBeenCalled();
  });
});
