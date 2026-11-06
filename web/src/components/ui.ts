// Shared styles and small formatters for the flow pages.
import { focusRing } from './Layout';

export const primary = `inline-flex min-h-14 w-full items-center justify-center gap-3 rounded-full bg-line px-6 font-display text-xl font-bold text-on-line shadow-[0_18px_36px_-18px_var(--line)] transition-transform active:scale-[0.98] disabled:opacity-60 sm:w-auto ${focusRing}`;
export const secondary = `inline-flex min-h-14 w-full items-center justify-center gap-2 rounded-full border-[1.5px] border-rule bg-card px-6 font-display text-lg font-bold text-ink transition-colors hover:border-line sm:w-auto ${focusRing}`;
export const textLink = `inline-flex min-h-11 items-center rounded font-bold text-line underline underline-offset-4 ${focusRing}`;
// A large number field: digits in the mono face, easy to hit and read in a moving train.
export const numberInput = `w-full rounded-2xl border-2 border-rule bg-card px-5 py-4 font-mono text-3xl font-bold tracking-[0.12em] text-ink placeholder:text-rule focus:border-line focus:outline-none ${focusRing}`;
export const card = `w-full rounded-3xl border-[1.5px] border-rule bg-card px-5 py-4 text-left transition-colors hover:border-line ${focusRing}`;

export function formatDistance(m: number, lang: string) {
  if (m >= 1000) return `${(m / 1000).toLocaleString(lang, { maximumFractionDigits: 1 })} km`;
  return `${Math.round(m / 10) * 10} m`;
}

// onlyDigits keeps what the user typed as a number: phones sometimes add spaces or letters.
export function onlyDigits(s: string) {
  return s.replace(/\D/g, '');
}
