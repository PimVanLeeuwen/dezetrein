import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { Link, navigate, parseRoute, paths, useRoute } from './router';

function Show() {
  const r = useRoute();
  return <p data-testid="route">{r.name === 'train' ? `train ${r.train}` : r.name}</p>;
}

afterEach(() => window.history.replaceState(null, '', '/'));

describe('parseRoute', () => {
  it.each([
    ['/', { name: 'home' }],
    ['/trein/2450', { name: 'train', train: '2450' }],
    ['/trein/303540/', { name: 'train', train: '303540' }],
    ['/trein/1234567', { name: 'notFound' }],
    ['/trein/abc', { name: 'notFound' }],
    ['/trein/', { name: 'notFound' }],
    ['/vind', { name: 'find' }],
    ['/privacy', { name: 'privacy' }],
    ['/over', { name: 'about' }],
    ['/nope', { name: 'notFound' }],
  ])('%s', (path, want) => expect(parseRoute(path)).toEqual(want));
});

describe('navigation', () => {
  it('updates the route on navigate, back and forward', () => {
    render(<Show />);
    expect(screen.getByTestId('route')).toHaveTextContent('home');
    act(() => navigate(paths.train('3522')));
    expect(screen.getByTestId('route')).toHaveTextContent('train 3522');
    expect(window.location.pathname).toBe('/trein/3522');
    act(() => {
      window.history.replaceState(null, '', '/privacy');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });
    expect(screen.getByTestId('route')).toHaveTextContent('privacy');
  });

  it('navigates in place on a plain click, and leaves modified clicks to the browser', () => {
    render(
      <>
        <Show />
        <Link to="/over">about</Link>
      </>,
    );
    const link = screen.getByRole('link', { name: 'about' });
    expect(link).toHaveAttribute('href', '/over');
    const ctrl = fireEvent.click(link, { ctrlKey: true });
    expect(ctrl).toBe(true); // not prevented: the browser opens a new tab
    expect(screen.getByTestId('route')).toHaveTextContent('home');
    fireEvent.click(link);
    expect(screen.getByTestId('route')).toHaveTextContent('about');
  });
});
