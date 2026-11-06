// A tiny router on the History API: the site has five pages, so a dependency is not worth it.
// nginx serves index.html for every unknown path, so a refresh or a shared link lands here.
import { useSyncExternalStore, type AnchorHTMLAttributes, type MouseEvent } from 'react';

export type Route =
  | { name: 'home' }
  | { name: 'train'; train: string }
  | { name: 'privacy' }
  | { name: 'about' }
  | { name: 'notFound' };

// Journey numbers are 1 to 6 digits, as the backend accepts.
const trainPath = /^\/trein\/([0-9]{1,6})\/?$/;

export function parseRoute(path: string): Route {
  if (path === '/' || path === '') return { name: 'home' };
  const train = trainPath.exec(path);
  if (train?.[1]) return { name: 'train', train: train[1] };
  if (path === '/privacy' || path === '/privacy/') return { name: 'privacy' };
  if (path === '/over' || path === '/over/') return { name: 'about' };
  return { name: 'notFound' };
}

export const paths = {
  home: '/',
  train: (n: string) => `/trein/${encodeURIComponent(n)}`,
  privacy: '/privacy',
  about: '/over',
};

// pushState fires no event, so navigate announces itself to the subscribers.
const changed = 'dezetrein:navigate';

export function navigate(to: string, { replace = false } = {}) {
  if (to === window.location.pathname) return;
  if (replace) window.history.replaceState(null, '', to);
  else window.history.pushState(null, '', to);
  window.dispatchEvent(new Event(changed));
  window.scrollTo?.(0, 0);
}

function subscribe(onChange: () => void) {
  window.addEventListener('popstate', onChange);
  window.addEventListener(changed, onChange);
  return () => {
    window.removeEventListener('popstate', onChange);
    window.removeEventListener(changed, onChange);
  };
}

export function useRoute(): Route {
  const path = useSyncExternalStore(subscribe, () => window.location.pathname);
  return parseRoute(path);
}

type LinkProps = AnchorHTMLAttributes<HTMLAnchorElement> & { to: string };

// Link is a normal anchor (so it opens in a new tab, and works without JavaScript on the
// server's fallback) that navigates in place on a plain left click.
export function Link({ to, onClick, ...rest }: LinkProps) {
  const handle = (e: MouseEvent<HTMLAnchorElement>) => {
    onClick?.(e);
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    navigate(to);
  };
  return <a href={to} onClick={handle} {...rest} />;
}
