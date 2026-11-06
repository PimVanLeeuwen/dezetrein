import { act, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { isDark, nextTheme, useTheme } from './theme';

let systemDark = false;
let listeners: (() => void)[] = [];

beforeEach(() => {
  systemDark = false;
  listeners = [];
  vi.stubGlobal('matchMedia', (q: string) => ({
    media: q,
    get matches() {
      return systemDark;
    },
    addEventListener: (_: string, l: () => void) => listeners.push(l),
    removeEventListener: (_: string, l: () => void) => (listeners = listeners.filter((x) => x !== l)),
  }));
});

afterEach(() => {
  window.localStorage.clear();
  document.documentElement.classList.remove('dark');
  vi.unstubAllGlobals();
});

function Probe() {
  const [theme, setTheme] = useTheme();
  return (
    <button type="button" onClick={() => setTheme(nextTheme(theme))}>
      {theme}
    </button>
  );
}

describe('theme', () => {
  it('resolves and cycles', () => {
    expect(isDark('system', true)).toBe(true);
    expect(isDark('light', true)).toBe(false);
    expect(isDark('dark', false)).toBe(true);
    expect([nextTheme('system'), nextTheme('light'), nextTheme('dark')]).toEqual(['light', 'dark', 'system']);
  });

  it('follows the system until the user picks, and remembers the pick', () => {
    render(<Probe />);
    const html = document.documentElement;
    expect(html).not.toHaveClass('dark');
    systemDark = true;
    act(() => listeners.forEach((l) => l()));
    expect(html).toHaveClass('dark');

    act(() => screen.getByRole('button').click()); // light
    expect(html).not.toHaveClass('dark');
    expect(window.localStorage.getItem('dezetrein.theme')).toBe('light');
    expect(listeners).toHaveLength(0); // no longer following the system

    act(() => screen.getByRole('button').click()); // dark
    systemDark = false;
    expect(html).toHaveClass('dark');

    act(() => screen.getByRole('button').click()); // back to system
    expect(html).not.toHaveClass('dark');
    expect(window.localStorage.getItem('dezetrein.theme')).toBeNull();
  });

  it('starts from the stored choice', () => {
    window.localStorage.setItem('dezetrein.theme', 'dark');
    render(<Probe />);
    expect(screen.getByRole('button')).toHaveTextContent('dark');
    expect(document.documentElement).toHaveClass('dark');
  });
});
