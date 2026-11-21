import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// The console is served by the binary, from the same origin as its assets, but it calls an API that
// may live on another host. The assets are absolute (`/assets/…`), not relative: the console has deep
// links of its own (`/targets/:ref`), and `./assets` under one of those resolves to
// `/targets/assets/…`, which the server answers with the shell — a page that loads and renders
// nothing.
export default defineConfig({
  base: '/',
  plugins: [react(), tailwindcss()],
  build: {
    // Straight into what the binary embeds: one command builds the console the image carries.
    outDir: '../internal/web/dist',
    emptyOutDir: true,
    // Monaco is the biggest thing here and only the editor screens need it.
    chunkSizeWarningLimit: 2048,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('monaco-editor') || id.includes('@monaco-editor')) return 'monaco';
          if (id.includes('node_modules')) return 'vendor';
          return undefined;
        },
      },
    },
  },
  server: {
    // `npm run dev` talks to the API on the constant port; the origin is in CONSOLE_ORIGINS.
    port: 5173,
  },
  // The unit tests are the console's pure parts: formatting a timestamp, keeping a draft, splitting a
  // diff, and the schemas a form runs on. jsdom is enough for the two that touch localStorage, and the
  // screens are checked by hand and by the browser pass in the ticket.
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
});
