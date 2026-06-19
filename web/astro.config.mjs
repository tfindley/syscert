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
  // Content-Security-Policy. Astro auto-generates + maintains the script-src/style-src
  // hashes for every inline script and <style> block on each build (so script edits
  // never silently break the policy), and adds these directives. Delivered via a <meta>
  // element — frame-ancestors can't be set there, so clickjacking is covered by the
  // X-Frame-Options: SAMEORIGIN header in web/nginx.conf. The site has no inline style
  // ATTRIBUTES (all moved to classes) and self-hosts its fonts, so 'self' suffices.
  security: {
    csp: {
      directives: [
        "default-src 'self'",
        "img-src 'self' data:",
        "font-src 'self'",
        "connect-src 'self'",
        "object-src 'none'",
        "base-uri 'self'",
      ],
    },
  },
  integrations: [sitemap()],
  vite: {
    plugins: [tailwindcss()],
  },
});
