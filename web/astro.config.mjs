// @ts-check
import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";

// Canonical site URL. Swap to https://syscert.tfindley.dev once DNS is live.
const SITE = process.env.SITE_URL || "https://syscert.tfindley.dev";

// https://astro.build/config
export default defineConfig({
  site: SITE,
  redirects: {
    "/docs": "/docs/quick-start/",
  },
  // Docs code blocks use the site's own .doc-prose styling (consistent with the
  // marketing pages), not Shiki's inline theme.
  markdown: {
    syntaxHighlight: false,
  },
  integrations: [sitemap()],
  vite: {
    plugins: [tailwindcss()],
  },
});
