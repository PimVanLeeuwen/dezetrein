import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { navigate } from './lib/router';

beforeEach(() => vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['nl-NL']));
afterEach(() => {
  window.history.replaceState(null, '', '/');
  window.localStorage.clear();
  document.documentElement.classList.remove('dark');
  vi.restoreAllMocks();
});

describe('App', () => {
  it('shows the start screen with both ways in', () => {
    render(<App />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Vind de trein waarin je zit');
    expect(screen.getByRole('button', { name: /Vind mijn trein/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Treinstelnummer/ })).toBeInTheDocument();
    expect(screen.getByText(/niet verbonden aan NS/)).toBeInTheDocument();
    expect(document.title).toBe('Vind de trein waarin je zit · dezetrein');
  });

  it('opens the find flow from the start screen', () => {
    render(<App />);
    fireEvent.click(screen.getByRole('button', { name: /Vind mijn trein/ }));
    expect(window.location.pathname).toBe('/vind');
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Waar ben je?');
  });

  it('switches to English and back', () => {
    render(<App />);
    fireEvent.click(screen.getByRole('button', { name: 'Switch to English' }));
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Find the train you are on');
    expect(document.documentElement.lang).toBe('en');
    fireEvent.click(screen.getByRole('button', { name: 'Wissel naar Nederlands' }));
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Vind de trein waarin je zit');
  });

  it('cycles the theme from the header', () => {
    render(<App />);
    const button = screen.getByRole('button', { name: /Thema: systeem/ });
    fireEvent.click(button);
    expect(screen.getByRole('button', { name: /Thema: licht/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Thema: licht/ }));
    expect(document.documentElement).toHaveClass('dark');
  });

  it.each([
    ['/trein/2450', 'Trein 2450'],
    ['/vind', 'Waar ben je?'],
    ['/privacy', 'Privacy'],
    ['/over', 'Over dezetrein'],
    ['/bestaat-niet', 'Pagina niet gevonden'],
  ])('renders %s', (path, heading) => {
    window.history.replaceState(null, '', path);
    render(<App />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(heading);
    expect(document.title).toBe(`${heading} · dezetrein`);
  });

  it('navigates from the footer and moves focus to the new heading', () => {
    render(<App />);
    const link = screen.getByRole('link', { name: 'Privacy' });
    link.focus();
    fireEvent.click(link);
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toHaveTextContent('Privacy');
    expect(h1).toHaveFocus();
    act(() => navigate('/'));
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Vind de trein waarin je zit');
  });
});
