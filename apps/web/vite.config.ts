import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";
import { defineConfig, loadEnv } from "vite";

// One predictable config: dist into <repo>/bin/dist (the Go embed sync), the
// API proxied for dev under /v1, no surprises across environments. `pnpm dev`
// talks to localhost:9000 by default; the compose dev service sets
// VITE_PROXY_TARGET to http://noxis:9000.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".");
  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
    },
    build: {
      outDir: "../../bin/dist",
      emptyOutDir: true,
      target: "es2020",
      sourcemap: false,
    },
    server: {
      host: true,
      port: 5173,
      proxy: {
        // the public API is mounted under /external; the SPA's /auth/callback
        // is a client route and must not be proxied.
        "/external": {
          target: env.VITE_PROXY_TARGET ?? "http://localhost:9000",
          changeOrigin: true,
        },
      },
    },
  };
});
