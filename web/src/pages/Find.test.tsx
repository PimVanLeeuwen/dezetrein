import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { FindState } from '../flow/useFindTrain';
import { I18nProvider } from '../i18n/i18n';
import { candidate, result } from '../test/fakes';
import { FindView } from './Find';

beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl']));
afterEach(() => {
  window.history.replaceState(null, '', '/');
  vi.restoreAllMocks();
});

const wrap = (ui: ReactNode) => render(<I18nProvider>{ui}</I18nProvider>);

function view(state: FindState) {
  const start = vi.fn();
  const reset = vi.fn();
  wrap(<FindView state={state} start={start} reset={reset} />);
  return { start, reset };
}

describe('FindView', () => {
  it('explains before asking for location', () => {
    const { start } = view({ phase: 'intro' });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Waar ben je?');
    expect(screen.getByText(/nooit bewaard/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Deel mijn locatie/ }));
    expect(start).toHaveBeenCalled();
  });

  it('shows progress while locating, and lets the user stop', () => {
    const { reset } = view({ phase: 'locating', readings: 3, accuracyM: 1414, checking: false });
    expect(screen.getByRole('status')).toHaveTextContent('3 metingen, nauwkeurig tot 1,4 km');
    expect(screen.getByText('Grove locatie, dat is genoeg.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Stoppen' }));
    expect(reset).toHaveBeenCalled();
  });

  it('says when it compares trains', () => {
    view({ phase: 'locating', readings: 2, accuracyM: 12, checking: true });
    expect(screen.getByRole('status')).toHaveTextContent('Treinen in de buurt vergelijken');
  });

  it('asks about the best train by destination, with its carriages, and offers the others', () => {
    view({ phase: 'confirm', result: result('low', [candidate('3539', 'Venlo', ['9574']), candidate('5639', 'Zwolle', ['2416', '2467']), candidate('8940')]) });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Zit je in de trein naar Venlo?');
    expect(screen.getByText('Intercity 3539')).toBeInTheDocument();
    expect(screen.getByLabelText('Treinstellen: 9574')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('Of zit je in een van deze?');
    expect(screen.getByRole('button', { name: /Zwolle/ })).toHaveTextContent('2416');
    expect(screen.getByRole('button', { name: /Intercity 8940/ })).toHaveTextContent('Bestemming onbekend');
    expect(screen.getByRole('button', { name: /Intercity 8940/ })).toHaveTextContent('Treinstellen onbekend');
  });

  it('goes to the trip on yes, or to another candidate', () => {
    view({ phase: 'confirm', result: result('low', [candidate('3539', 'Venlo'), candidate('5639', 'Zwolle')]) });
    fireEvent.click(screen.getByRole('button', { name: /Ja, deze trein/ }));
    expect(window.location.pathname).toBe('/trein/3539');
    fireEvent.click(screen.getByRole('button', { name: /Zwolle/ }));
    expect(window.location.pathname).toBe('/trein/5639');
  });

  it('searches again on no', () => {
    const { start } = view({ phase: 'confirm', result: result('high', [candidate('3539', 'Venlo')]) });
    fireEvent.click(screen.getByRole('button', { name: 'Nee, zoek opnieuw' }));
    expect(start).toHaveBeenCalled();
  });

  it('asks by number when the destination is unknown', () => {
    view({ phase: 'confirm', result: result('high', [candidate('3539')]) });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Zit je in trein 3539?');
  });

  it.each([
    ['denied', 'Je locatie staat uit', true],
    ['timeout', 'Geen locatie gevonden', true],
    ['none', 'Geen trein gevonden', true],
    ['outside', 'Je bent buiten het spoornet', false],
    ['unsupported', 'Locatie werkt hier niet', false],
  ] as const)('explains %s', (reason, title, canRetry) => {
    view({ phase: 'failed', reason });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(title);
    expect(screen.queryByRole('button', { name: 'Probeer opnieuw' }) !== null).toBe(canRetry);
    expect(screen.getByRole('link', { name: 'Naar de start' })).toHaveAttribute('href', '/');
  });

  it('shows the API error message', () => {
    view({ phase: 'failed', reason: 'api', message: 'error.rateLimited' });
    expect(screen.getByText(/te veel verzoeken/)).toBeInTheDocument();
  });
});
