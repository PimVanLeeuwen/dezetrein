import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import App from './App';

describe('App', () => {
  it('shows the name and what the site does, in Dutch and English', () => {
    render(<App />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('dezetrein');
    expect(screen.getByText(/Vind de trein waarin je zit/)).toBeInTheDocument();
    expect(screen.getByText(/Find the train you are on/)).toHaveAttribute('lang', 'en');
  });
});
