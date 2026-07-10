import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import relay from "vite-plugin-relay";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    react({
      babel: {
        plugins: ["relay"],
      },
    }),
    relay,
  ],
  build: {
    // Must land under internal/server for //go:embed in web_release.go
    outDir: "../internal/server/web/dist",
    emptyOutDir: true,
  },
  server: {
    // Match internal/server/web_dev.go reverse proxy target.
    host: "127.0.0.1",
    port: 5174,
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
});
