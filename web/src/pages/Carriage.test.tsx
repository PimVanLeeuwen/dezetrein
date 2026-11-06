import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import type { CarriageResult, Fix } from '../api/types';
import { I18nProvider } from '../i18n/i18n';
import { fakeGeo } from '../test/fakes';
import { Carriage } from './Carriage';

beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl']));
afterEach(() => {
  window.history.replaceState(null, '', '/');
  vi.restoreAllMocks();
});

const found = (train: string, located = false): CarriageResult => ({ carriage: '9402', train: { train, carriages: ['9402'] }, located, searched: located ? 4 : 0, stale: false });
const missing = (located: boolean, searched = 0): CarriageResult => ({ carriage: '9402', train: null, located, searched, stale: false });

function setup(...answers: (CarriageResult | Error)[]) {
  const g = fakeGeo();
  const calls: Fix[][] = [];
  const client = {
    carriage: vi.fn(async (_c: string, fixes: Fix[]) => {
      calls.push(fixes);
      const a = answers[Math.min(calls.length - 1, answers.length - 1)]!;
      if (a instanceof Error) throw a;
      return a;
    }),
  };
  render(
    <I18nProvider>
      <Carriage client={client} geo={g.geo} />
    </I18nProvider>,
  );
  return { g, client, calls };
}

const type = (v: string) => {
  fireEvent.change(screen.getByLabelText('Treinstelnummer'), { target: { value: v } });
  fireEvent.click(screen.getByRole('button', { name: /Zoek deze trein/ }));
};

describe('Carriage search', () => {
  it('goes straight to the trip when the trainset was seen recently, without location', async () => {
    const { client } = setup(found('3539'));
    await act(async () => type(' 94 02 '));
    expect(client.carriage).toHaveBeenCalledWith('9402', [], expect.any(AbortSignal));
    expect(window.location.pathname + window.location.search).toBe('/trein/3539?rijtuig=9402');
  });

  it('rejects anything but up to six digits, before any request', () => {
    const { client } = setup(found('1'));
    type('abc');
    expect(screen.getByText('Typ alleen cijfers, maximaal 6.')).toBeInTheDocument();
    type('1234567');
    expect(client.carriage).not.toHaveBeenCalled();
  });

  it('asks for location when the trainset is not known yet, then checks the trains nearby', async () => {
    const { g, calls } = setup(missing(false), found('5639', true));
    await act(async () => type('9402'));
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Waar ben je ongeveer?');
    expect(screen.getByText(/nooit bewaard/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Gebruik mijn locatie/ }));
    expect(screen.getByRole('status')).toHaveTextContent('Treinen in de buurt bekijken');
    await act(async () => {
      g.fix({ lat: 52.1 });
      g.fix({ lat: 52.2 });
    });
    expect(calls[1]).toHaveLength(2);
    expect(window.location.search).toBe('?rijtuig=9402');
    expect(g.watching()).toBe(false);
  });

  it('says when no train nearby has it, and offers other ways', async () => {
    const { g } = setup(missing(false), missing(true, 7));
    await act(async () => type('9402'));
    fireEvent.click(screen.getByRole('button', { name: /Gebruik mijn locatie/ }));
    await act(async () => {
      g.fix();
      g.fix();
    });
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Treinstel 9402 niet gevonden');
    expect(screen.getByText(/Geen van de 7 treinen/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Andere manieren' })).toHaveAttribute('href', '/zoek');
    fireEvent.click(screen.getByRole('button', { name: 'Ander nummer' }));
    expect(screen.getByLabelText('Treinstelnummer')).toBeInTheDocument();
  });

  it('explains a denied location and API errors', async () => {
    const a = setup(missing(false));
    await act(async () => type('9402'));
    fireEvent.click(screen.getByRole('button', { name: /Gebruik mijn locatie/ }));
    await act(async () => a.g.fail('denied'));
    expect(screen.getByText(/geeft je locatie niet door/)).toBeInTheDocument();
  });

  it('shows an API error', async () => {
    setup(new ApiError(429, 'rate_limited'));
    await act(async () => type('9402'));
    expect(screen.getByText(/te veel verzoeken/)).toBeInTheDocument();
  });
});
