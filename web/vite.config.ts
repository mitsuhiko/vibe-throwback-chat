import { defineConfig } from "vite";
import solidPlugin from "vite-plugin-solid";
import tailwindcss from "@tailwindcss/vite";
// import { consoleForwardPlugin } from "vite-console-forward-plugin";

export default defineConfig({
  plugins: [solidPlugin(), tailwindcss()], // consoleForwardPlugin() - commented out for Vite 6 compatibility
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
