import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { paraglideVitePlugin } from "@inlang/paraglide-js";
import path from "path";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: {
      $lib: path.resolve("./src/lib"),
    },
  },
  plugins: [
    tailwindcss(),
    paraglideVitePlugin({
      project: "./project.inlang",
      outdir: "./src/lib/paraglide",
      strategy: ["localStorage", "preferredLanguage", "baseLocale"],
    }),
    svelte(),
    wails("./bindings"),
  ],
  optimizeDeps: {
    entries: ["index.html", "config-editor.html"],
    include: [
      "@wailsio/runtime",
      "@codemirror/commands",
      "@codemirror/lang-json",
      "@codemirror/language",
      "@codemirror/lint",
      "@codemirror/state",
      "@codemirror/view",
      "@lucide/svelte",
      "@sv-kit/a11y-keys",
    ],
  },
  build: {
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, "index.html"),
        configEditor: path.resolve(__dirname, "config-editor.html"),
      },
    },
  },
});
