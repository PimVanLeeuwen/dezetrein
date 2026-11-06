import { useCallback, useEffect, useState } from 'react';

// The theme follows the system unless the user picks light or dark; the choice is remembered.
// public/theme.js applies the same rule before the first paint, so there is no flash.
export type Theme = 'system' | 'light' | 'dark';

export const themeStorageKey = 'dezetrein.theme';
export const themeOrder: readonly Theme[] = ['system', 'light', 'dark'];

const darkQuery = '(prefers-color-scheme: dark)';

export function readTheme(): Theme {
  try {
    const t = window.localStorage.getItem(themeStorageKey);
    if (t === 'light' || t === 'dark') return t;
  } catch {
    // storage blocked
  }
  return 'system';
}

export function isDark(theme: Theme, systemDark: boolean): boolean {
  return theme === 'dark' || (theme === 'system' && systemDark);
}

export function nextTheme(theme: Theme): Theme {
  return themeOrder[(themeOrder.indexOf(theme) + 1) % themeOrder.length] ?? 'system';
}

export function useTheme(): [Theme, (t: Theme) => void] {
  const [theme, setThemeState] = useState<Theme>(readTheme);
  useEffect(() => {
    const mq = window.matchMedia?.(darkQuery);
    const apply = () => document.documentElement.classList.toggle('dark', isDark(theme, mq?.matches ?? false));
    apply();
    if (theme !== 'system' || !mq) return;
    mq.addEventListener('change', apply);
    return () => mq.removeEventListener('change', apply);
  }, [theme]);
  const setTheme = useCallback((t: Theme) => {
    setThemeState(t);
    try {
      if (t === 'system') window.localStorage.removeItem(themeStorageKey);
      else window.localStorage.setItem(themeStorageKey, t);
    } catch {
      // not remembered, but still switched
    }
  }, []);
  return [theme, setTheme];
}
