# syscert website — TODO

Everything left to get the site online and finished. Generated 2026-06-14.

- **Where:** `~/devel/syscert/web/` (in the monorepo)
- **Domain:** `syscert.tfindley.dev`
- **Hosting:** Docker image on GHCR → run on Hetzner behind Traefik (Traefik does TLS)
- **Image:** `ghcr.io/tfindley/syscert-web:latest`
- **Design spec:** `web/docs/superpowers/specs/2026-06-13-syscert-web-design.md`

---

## ▶ Review the site locally

```sh
cd ~/devel/syscert/web
npm install        # first time only (or after a fresh clone)
npm run dev        # start the dev server
```

Then open **http://localhost:4321/** in your browser. (On WSL2, `localhost` forwards to Windows, so
it just works.) Stop the server with **Ctrl+C**.

Look at `/` and `/install/`. The nav **Docs/FAQ** links 404 for now — the docs section isn't built yet.

Prefer the exact production build (what actually ships in the container)?

```sh
npm run build && npm run preview   # also serves on http://localhost:4321
```

---

## 0. Decisions to make first

- [ ] **Website license** — the tool is AGPL-3.0; the site can match it or use MIT. Decide, then
      add a `LICENSE` (and footer already links it).
- [ ] **Auto-redeploy strategy** — Watchtower (a label is already set in `docker-compose.yml`) vs an
      SSH "deploy" step added to `.github/workflows/web.yml`. (Fine to skip and pull manually at first.)

---

## 1. Get it online

### Domain & DNS
- [ ] Register **`tfindley.dev`**.
- [ ] Add DNS **A** (and **AAAA** if you have IPv6) record: `syscert.tfindley.dev` → Hetzner box IP.
- [ ] Note: `.dev` is HSTS-preloaded (HTTPS-only). Traefik must present a valid cert before first visit
      — make sure the certresolver works for this host.

### Commit & push (this triggers the image build)
> Your syscert tree has uncommitted Go WIP — keep the web commit separate.
- [ ] Review the gitignore diff (it has your earlier change + my root-anchor change):
      ```sh
      cd ~/devel/syscert
      git diff .gitignore
      ```
- [ ] Stage only the website + CI changes and commit:
      ```sh
      git add web .github/workflows/web.yml .github/workflows/ci.yml .gitignore
      git commit -m "Add syscert website (Astro) + container build → GHCR"
      git push
      ```
- [ ] Confirm the **Web** GitHub Action ran and pushed `ghcr.io/tfindley/syscert-web:latest`.

### Registry access
- [ ] Make the **GHCR package public** (repo/owner → Packages → `syscert-web` → Package settings →
      Change visibility), **or** `docker login ghcr.io` on the Hetzner box with a PAT that has
      `read:packages`.

### Deploy on Hetzner (behind your Traefik)
- [ ] Put `web/docker-compose.yml` on the box (or merge into your existing stack).
- [ ] Edit it to match **your** Traefik:
  - [ ] external network name (file uses `traefik`)
  - [ ] certresolver name (file uses `letsencrypt`)
- [ ] Confirm your Traefik has a `websecure` entrypoint + a working ACME certresolver.
- [ ] Bring it up:
      ```sh
      docker compose pull && docker compose up -d
      ```
- [ ] (Optional) confirm your Traefik has a global HTTP→HTTPS redirect.

### Verify live
- [ ] `https://syscert.tfindley.dev` loads with a valid certificate.
- [ ] `/install/` renders; `/install.sh` returns the script (Content-Type `text/x-shellscript`).
- [ ] A bad URL (e.g. `/nope`) shows the styled **404**.
- [ ] Theme toggle persists; **copy** button on the install command works.
- [ ] `/sitemap-index.xml` and `/robots.txt` resolve.
- [ ] Quick Lighthouse / mobile check.

---

## 2. Finish the site (remaining build work)

> Happy to do these with you — just ask.

- [x] **Docs section** — `quick-start`, `configuration`, `advanced-install`, `distributing`,
      `troubleshooting`, `faq`. Built as hand-authored Astro pages under `src/pages/docs/` with a
      shared `DocsLayout.astro` (sidebar + prose), **not** Starlight (the scaffold has no Starlight
      integration). `/docs` redirects to `/docs/quick-start/`. Nav/footer no longer 404.
- [ ] **`web/CLAUDE.md`** — the original `/init` deliverable, written against the real scaffold.
- [ ] **OG social image** — create `web/public/og.png` (referenced by `<head>` for link previews).
- [ ] **Self-host fonts** — swap Google Fonts CDN for Fontsource packages (perf + privacy + offline
      builds). Fonts: Martian Mono, IBM Plex Sans, IBM Plex Mono.
- [ ] **Live version badge** — `src/consts.ts` has a static `version: "v0.0.6"`; wire it to fetch the
      latest GitHub release at build time (with the static value as fallback).
- [x] **`net-install.sh`** (in the **syscert repo**, `packaging/`) — the real one-line installer:
      detect OS/arch → resolve the latest release tag → download the matching binary → verify sha256
      against `sha256sums.txt` → fetch the systemd packaging at that tag → delegate to
      `packaging/install.sh`. Vendored to `web/public/install.sh` by the npm `prebuild` step
      (`scripts/vendor-install.mjs`); the committed copy is what the `context: ./web` Docker image
      ships, so re-run `npm run build` (or the script) and commit `public/install.sh` after editing
      the canonical script. **Not yet end-to-end tested against a live release/DNS.**
- [ ] **Content review** — read every claim for accuracy: compatibility list, benefits, the homepage
      `syscert.toml`/`secrets` snippet, and the command descriptions.

---

## 3. Housekeeping & follow-ups

- [ ] `npm audit` reports **3 high-severity** issues — review (don't blind `--force`).
- [ ] Delete the old leftover: `rm -rf ~/devel/syscert_web` (now just stale `.claude/`/`.superpowers/`).
- [ ] Optional: add a **CSP** header in `nginx.conf` (omitted for now — inline theme script + Google
      Fonts; needs nonces/hashes or self-hosted fonts first).
- [ ] Optional: privacy-friendly **analytics**.

---

## Reference

| Thing | Value |
| --- | --- |
| Local dev | `cd ~/devel/syscert/web && npm run dev` |
| Local prod preview | `npm run build && npm run preview` |
| Screenshots | `node scripts/shot.mjs` (Playwright) |
| Image | `ghcr.io/tfindley/syscert-web:latest` (+ short-sha) |
| Install one-liner | `curl -fsSL https://syscert.tfindley.dev/install.sh \| sudo sh` |
| Links source | `src/consts.ts` (GitHub, Issues, Releases, Ko-fi, site, version) |
| Workflows | `.github/workflows/web.yml` (image) · `ci.yml` (Go, ignores `web/**`) |

**Stack:** Astro 6 + Tailwind v4, native `<script>` islands. Dark "Terminal Trust" theme
(Martian Mono + IBM Plex), OS-default + persisted toggle. Note: `package.json` pins
`"overrides": { "vite": "^7" }` — required, or `astro build` breaks.
