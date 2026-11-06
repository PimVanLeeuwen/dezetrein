import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Testing Library unmounts automatically only with Vitest globals on; they are off here.
afterEach(cleanup);
