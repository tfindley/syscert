// Central site metadata + canonical links. One source of truth for nav/footer/SEO.

export const SITE = {
  name: "syscert",
  domain: "syscert.tfindley.dev",
  tagline: "Set-and-forget TLS for every machine.",
  description:
    "syscert is a small, least-privilege Linux service that gives every host its own auto-renewing TLS certificate, from Let's Encrypt, HashiCorp Vault, or Smallstep step-ca, and delivers it to local consumers with the exact ownership, mode, and SELinux context each one needs. One static binary and a systemd timer. No cron, no renewal scripts to babysit — on one host, or across a fleet with the Ansible role.",
  status: "early · pre-1.0",
  version: "v0.4.0", // static fallback; the live value comes from GitHub Releases at build time (src/lib/release.ts)
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

// Install routes, shown on /install and in the footer. Keep in step with
// docs/advanced-install.md — that page is the canonical list.
export const INSTALL_ROUTES = [
  { label: "One-line installer", href: "/install/" },
  { label: "Manual / verified binary", href: "/docs/advanced-install/manually/" },
  { label: "Offline (air-gapped)", href: "/docs/advanced-install/offline/" },
  { label: "Ansible (fleet)", href: "/docs/advanced-install/ansible/" },
  { label: "Compile from source", href: "/docs/advanced-install/compile-from-source/" },
] as const;

// The docs sidebar is derived from the docs content collection (sorted by
// frontmatter `order`) in DocsLayout — see web/src/layouts/DocsLayout.astro.
