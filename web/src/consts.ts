// Central site metadata + canonical links. One source of truth for nav/footer/SEO.

export const SITE = {
  name: "syscert",
  domain: "syscert.tfindley.dev",
  tagline: "Set-and-forget TLS for every machine.",
  description:
    "syscert is a small, least-privilege Linux service that gives every host its own auto-renewing TLS certificate — from Let's Encrypt, HashiCorp Vault, or Smallstep step-ca — and delivers it to local consumers with the exact ownership, mode, and SELinux context each needs. One static binary and a systemd timer. No cron, no scripts, no cert babysitting.",
  status: "early · pre-1.0",
  version: "v0.0.6", // build-time fetch from GitHub Releases later; static fallback here
} as const;

export const INSTALL_CMD = `curl -fsSL https://${SITE.domain}/install.sh | sudo sh`;

export const LINKS = {
  github: "https://github.com/tfindley/syscert",
  issues: "https://github.com/tfindley/syscert/issues",
  releases: "https://github.com/tfindley/syscert/releases",
  security: "https://github.com/tfindley/syscert/blob/main/SECURITY.md",
  license: "https://github.com/tfindley/syscert/blob/main/LICENSE",
  kofi: "https://ko-fi.com/tfindley",
  author: "https://tfindley.co.uk",
} as const;

export const NAV = [
  { label: "Docs", href: "/docs/quick-start/" },
  { label: "Install", href: "/install/" },
  { label: "FAQ", href: "/docs/faq/" },
] as const;

// The docs sidebar is derived from the docs content collection (sorted by
// frontmatter `order`) in DocsLayout — see web/src/layouts/DocsLayout.astro.
