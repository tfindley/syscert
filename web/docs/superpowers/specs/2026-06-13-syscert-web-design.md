# syscert website — design

**Date:** 2026-06-13
**Status:** Approved (brainstorm complete) — ready for implementation planning
**Project dir:** `/home/tristan/devel/syscert_web`
**Tool repo:** `/home/tristan/devel/syscert` (`github.com/tfindley/syscert`, AGPL-3.0, currently `v0.0.6`, pre-1.0)

---

## 1. Goal

A modern, fast, SEO-optimised marketing + documentation website for **syscert** — a CLI/systemd
service that gives every Linux host its own auto-renewing TLS certificate. The site must:

- Explain the tool in simple terms on the front page and say *why it's worth using*.
- Present a one-line install command with a one-click copy button (Homebrew / oh-my-zsh style).
- Link to alternative install options right next to the one-liner.
- Provide full documentation, FAQ, troubleshooting, and advanced install behind the homepage.
- Work on desktop and mobile, be accessible, and rank well.

The final deliverable of the implementation that follows this spec also includes a **`CLAUDE.md`**
for this repo (the original `/init` request), written against the real scaffold.

## 2. What syscert is (for accurate copy)

"Set-and-forget TLS for every machine." A single static Go binary plus a systemd timer. Each host
gets its own certificate from **Let's Encrypt**, **HashiCorp Vault**, or **Smallstep `step-ca`**,
keeps it renewed, and delivers it to local consumers (nginx, HAProxy, Cockpit, Postgres, Redis…)
with the exact ownership, mode, and SELinux context each needs. No cron, no scripts. Speaks ACME via
lego; writes certbot-compatible output (`cert.pem` / `privkey.pem` / `chain.pem` / `fullchain.pem`)
plus an all-in-one `bundle.pem`. Independent of any host `certbot`.

Commands to surface: `ensure` (default) · `issue` · `renew` · `distribute` · `void` · `destroy` ·
`dry-run` · `trust install`/`remove`. Targets: Debian/Ubuntu and RHEL family; amd64 + arm64.

## 3. Locked decisions

| Area | Decision |
| --- | --- |
| Repo location | **Build in this standalone `syscert_web` repo for now.** Structure it so it can later drop into `syscert/web/` (monorepo) with no rework. |
| Tech stack | **Astro** (static output) for marketing pages + **Starlight** for `/docs`; **Tailwind**; **React + shadcn/ui islands** used sparingly. |
| Hosting | **GitHub Pages** via **GitHub Actions**, custom domain. |
| Domain | **`syscert.tfindley.dev`** (requires registering `tfindley.dev`). Clean apex → no base-path. |
| Install UX | New **`curl … | sudo sh`** network installer at the apex, plus prominent inspect-first / checksum / binary-download alternatives. |
| Visual direction | **Terminal Trust (dark)** — near-black `#0b0e14`, cyan `#22d3ee`, green `#34d399`, monospace accents. |
| Theme | Default to host OS (`prefers-color-scheme`); manual toggle persisted in `localStorage`; applies to homepage **and** docs. |

## 4. Architecture

### 4.1 Directory layout (standalone now, monorepo-ready)

```
syscert_web/                      # later: syscert/web/
├── astro.config.mjs              # site, base "/", sitemap, Starlight integration
├── src/
│   ├── pages/                    # Astro marketing pages
│   │   ├── index.astro           # homepage
│   │   └── install.astro         # install hub
│   ├── content/docs/             # Starlight MDX docs collection (incl. faq — nav "FAQ" → /docs/faq)
│   ├── components/
│   │   ├── CopyCommand.tsx        # React island — command + copy button + "copied" state
│   │   ├── ThemeToggle.tsx        # React island — OS default + persisted override
│   │   ├── Hero.astro, Benefits.astro, Compatibility.astro, HowItWorks.astro,
│   │   ├── Footer.astro, VersionBadge.astro, SupportCTA.astro
│   ├── styles/                    # Tailwind entry + theme tokens
│   └── lib/                       # build-time helpers (latest release, config-ref transform)
├── public/
│   ├── install.sh                # vendored copy of the network installer (served at apex)
│   ├── robots.txt, sitemap (generated), favicon, og-image
│   └── CNAME                      # syscert.tfindley.dev
└── .github/workflows/deploy.yml  # build + deploy to Pages
```

### 4.2 Stack rationale

- **Astro** ships ~zero JS for static content → strong Core Web Vitals / SEO. Interactivity is
  opt-in via **islands**, so only `CopyCommand` and `ThemeToggle` (and anything genuinely
  interactive) load JS.
- **Starlight** provides the docs shell: left sidebar, **Pagefind** static search (free), MDX,
  prev/next, "edit this page" links, and a matching theme.
- **Tailwind** for styling; a small token layer encodes the Terminal-Trust palette and is the single
  source for both Astro and React-island components.
- **shadcn/ui** remains available (MCP wired) for any richer component; used sparingly to avoid
  bloat — default to plain Astro markup.

### 4.3 Theme system

CSS custom properties for the palette; `:root` = dark tokens, `[data-theme="light"]` = light
tokens. An inline, render-blocking head snippet sets `data-theme` from `localStorage` else
`matchMedia('(prefers-color-scheme: dark)')` **before paint** (no flash). `ThemeToggle` flips and
persists. Both homepage and docs consume the same tokens.

## 5. Information architecture

```
/              Homepage (marketing)
/install       Install hub (one-liner + inspect-first + binary + source + checksums + uninstall)
/docs/…        Starlight:
                 quick-start          (install → edit two files → done)
                 configuration        (the [cert]/[acme]/[acme.dns]/[store]/[bundle]/[[distribute]] reference)
                 advanced-install     (download binary, build from source, manual systemd steps)
                 distributing         (delivering certs to consumers; perms/SELinux; no auto-reload, ADR-0019)
                 troubleshooting
                 faq
External:      GitHub ↗ · Releases ↗ · Issues ↗   (links, not maintained pages)
```

## 6. Homepage spec (top → bottom)

1. **Sticky nav** — logo, Docs, Install, FAQ, GitHub ↗, theme toggle.
2. **Hero** — headline "Set-and-forget TLS for every machine."; one-sentence subhead; the install
   command in a terminal-styled box with **CopyCommand**; secondary CTAs **[ Other install options ]**
   and **[ Read the docs ]**; a trust line: **Inspect the script first ↗** · verify checksums.
   Small **"early · pre-1.0"** status badge. **Latest: vX.Y.Z →** version indicator (build-time fetch).
3. **Why / problem → solution** — the edge→backend & service→service plaintext problem in 2–3
   sentences; one simple diagram.
4. **How it works — 3 steps** — (1) Install one binary + timer; (2) Edit two files; (3) Done, renews
   forever. Step 2 shows a **real minimal config**:

   ```toml
   # /etc/syscert/syscert.toml
   [cert]
   hostname = "host.example.com"
   sans     = ["api.example.com"]

   [acme]
   ca        = "letsencrypt"
   email     = "you@example.com"
   challenge = "dns-01"

   [acme.dns]
   provider = "cloudflare"

   [[distribute]]
   artifact = "fullchain"
   path     = "/etc/nginx/tls/fullchain.pem"
   owner    = "root"
   group    = "nginx"
   mode     = "0644"
   ```
   ```sh
   # /etc/syscert/secrets   (env file, 0640 — DNS/CA creds, never in the .toml)
   CLOUDFLARE_DNS_API_TOKEN=…
   ```
5. **Benefits grid** — no cron/scripts · least-privilege service · correct owner/mode/SELinux ·
   certbot-compatible output · mTLS between services · multi-CA.
6. **Compatibility** — CAs (Let's Encrypt · Vault · step-ca) · consumers (nginx · HAProxy · Cockpit ·
   Postgres · Redis) · OS (Debian/Ubuntu · RHEL family) · arch (amd64 · arm64).
7. **Features / commands** — the command list as a compact reference.
8. **Support this project** — a tasteful Ko-fi CTA (`ko-fi.com/tfindley`) above the footer.
9. **Final CTA** — install command again · ⭐ Star on GitHub · Read the docs.
10. **Footer** — grouped links:
    - **Project:** GitHub · Issues · Releases · Security policy
    - **Docs:** Quick start · Configuration · Troubleshooting · FAQ
    - **Support:** ☕ Ko-fi (`ko-fi.com/tfindley`) · tfindley.co.uk
    - **Legal:** License (AGPL-3.0) · pre-1.0 notice

## 7. Install page spec (`/install`)

Leads with the one-liner, then **earns trust** (important: it's a security tool piped to root):

- **One-liner:** `curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh` (CopyCommand).
- **Inspect before you pipe:** show/download the script first; explain what it does.
- **Verify checksums:** the `sha256sums.txt` flow from the release.
- **Download a release binary** (amd64/arm64) + run `packaging/install.sh`.
- **Build from source** (Go ≥ 1.26).
- **Uninstall:** `sudo packaging/install.sh --uninstall [--purge]`.
- Supported targets + the pre-1.0 note.

## 8. Docs spec (Starlight)

User-facing and curated (the tool's `docs/` are engineering notes — not auto-imported). The
**configuration reference** is the one page worth single-sourcing: generate/validate it at build
from the canonical `config.sample.toml` + `docs/config-reference.md` in the syscert repo so it can't
drift. Other pages are written for users, linking back to the README for exhaustive detail.

## 9. Network installer (`net-install.sh`) — cross-repo deliverable

The hero command needs a script that does not exist yet (today's `packaging/install.sh` installs
from an already-downloaded binary; per ADR-0034 the installer is external to the binary by design).

**Design:** detect OS (Linux) + arch (amd64/arm64) → download the matching binary from the latest
GitHub Release → **verify sha256** against the release's `sha256sums.txt` (abort on mismatch) → fetch
the systemd packaging → delegate to the existing `packaging/install.sh`. Idempotent; requires root
(designed to be run via `| sudo sh`, or re-execs itself with sudo). Clear, readable, auditable
(people *will* read it before piping).

**Home & serving:** canonical source lives in the **syscert repo** (`packaging/net-install.sh`) so it
ships and is tested with the tool. The site's build **vendors a copy** into `public/install.sh`,
served at `https://syscert.tfindley.dev/install.sh`. *Cross-repo seam:* when the site later merges
into the monorepo (`syscert/web/`), the vendored copy is replaced by a direct reference — note left
for that migration. Until then the deploy workflow pins the version it vendors.

## 10. SEO / performance / accessibility

- Per-page `<title>`/description, Open Graph + Twitter card, generated social image.
- `sitemap.xml` (Astro integration), `robots.txt`, canonical URLs on `syscert.tfindley.dev`.
- JSON-LD `SoftwareApplication` structured data.
- Semantic headings, skip-link, keyboard-navigable nav, visible focus, AA contrast (palette checked),
  `prefers-reduced-motion` honored.
- Static output + system font stack → fast on mobile; lazy-load any heavy asset.

## 11. Components inventory

`CopyCommand` (island), `ThemeToggle` (island), `Hero`, `HowItWorks`, `Benefits`, `Compatibility`,
`Features`, `SupportCTA`, `VersionBadge` (build-time GitHub Release fetch with a static fallback),
`Footer`. Keep each component single-purpose.

## 12. Deploy pipeline

`.github/workflows/deploy.yml`: on push to `main` → set up Node → `npm ci` → `astro build` (vendoring
`install.sh`) → deploy `dist/` to GitHub Pages. `public/CNAME` = `syscert.tfindley.dev`; "Enforce
HTTPS" on. Requires a GitHub repo for the site and a DNS record once the domain is registered.

## 13. Scope

**In v1:** homepage, `/install`, the six docs pages, `net-install.sh` + vendoring, SEO essentials,
theme system, responsive + accessible, deploy workflow + CNAME, and the repo **`CLAUDE.md`**.

**Deferred (YAGNI):** blog/changelog (link to GitHub Releases), versioned docs (pre-1.0), i18n,
analytics, testimonials/logo wall, newsletter.

## 14. Open items / user dependencies

- Register **`tfindley.dev`**; add DNS for `syscert.tfindley.dev` → GitHub Pages.
- Create the GitHub repo for the site; enable Pages.
- **Site license** — pick the license for the *website* code/content (tool is AGPL-3.0; the site can
  match or use MIT). Decide before launch.
- `net-install.sh` lands in the **syscert repo** — coordinate that PR alongside the site.
- Mention on the site that the **Ansible role is not built yet** (set expectations; from README).

## 15. Risks / notes

- `curl | sudo sh` distrust → mitigated by inspect-first + checksums presented prominently.
- Version indicator must fail safe (static fallback) if the GitHub API is unavailable at build.
- Keep JS minimal — every island is a deliberate choice, not a default.
- Config-reference single-sourcing must degrade gracefully if the syscert repo isn't checked out at
  build time (fallback to a committed snapshot).
