import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Same origin as in production, where nginx proxies /api to the backend.
    proxy: { '/api': 'http://localhost:8080' },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    // The contrast test reads the palette from index.css (as ?raw); other CSS stays skipped.
    css: { include: [/index\.css/] },
    coverage: { reporter: ['text', 'json-summary', 'html'], include: ['src/**'], exclude: ['src/test/**', 'src/main.tsx'] },
  },
});
