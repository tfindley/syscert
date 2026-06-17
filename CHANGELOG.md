# Changelog

All notable changes to SysCert are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/), commit history follows
[Conventional Commits](https://www.conventionalcommits.org/), and versions follow
[Semantic Versioning](https://semver.org/).

`scripts/prerelease.sh` inserts a new version section below this line from the
Conventional Commits since the last tag; fill in the **Risk & Security** note
before publishing with `scripts/release.sh`.

<!-- next-release -->
## [v0.3.0] — 2026-06-18

### Features

- **cli:** `syscert status` — a read-only, offline snapshot of the resolved config,
  the stored certificate's subject/SANs/issuer/key and its issue, expiry, and
  renewal dates, the ACME account(s), archived snapshots, and distribution targets.
  No network, no credentials; it never prints the EAB HMAC. `ensure` now logs a
  one-line cert summary on completion, so the cert's expiry/renewal surface in
  `systemctl status syscert` / `journalctl -u syscert`.
- **cli:** `destroy --keep-account` removes only the certificate (and any archived
  snapshots) while keeping the ACME account, so the next run reissues **reusing the
  existing account — no fresh EAB token needed**. It also skips the internal-CA
  trust-anchor removal (you're reissuing, not tearing down trust).
- **store:** optional certificate history — `[store].archive_keep = N` snapshots the
  previous artifacts under `archive/<UTC>/` before each renewal (default `0`,
  disabled) — and configurable store-directory permissions, `[store].dir_mode` and
  `[store].group`, to grant a consumer group access. Key-bearing files stay `0600`
  regardless; only the directory mode/group are configurable.
- **web:** a collapsible docs submenu with dedicated per-config Sample Config pages,
  and a mobile hamburger menu for the top nav.

### Fixes

- **acme:** resolve the existing ACME account instead of re-registering on every run.
  A persisted account is now re-established with `ResolveAccountByKey` (no EAB), so a
  single-use External Account Binding token is validated only on first registration
  and never replayed — which makes a Vault `eab_policy = "always-required"` safe.

### Documentation

- New **EAB** page with a **Vault** subpage (request/list/revoke an EAB token, the
  `vault-eab-0-` format, single-use semantics) and a **Reloading services** guide for
  the `systemd.path` watch-and-reload pattern; documented the new `status` and
  `destroy --keep-account` commands and the `[store]` options.

### Risk & Security

Low risk. Certificate issuance and the privilege model are unchanged, and the
defaults preserve the existing `0700`, syscert-owned store with history disabled.
A `/security-review` of `v0.2.0..HEAD` found no vulnerabilities.

- **EAB handling is improved, not widened.** Resolving (rather than re-registering)
  the account means a single-use EAB token is sent only on first registration and
  never replayed; a non-`accountDoesNotExist` resolve error fails closed rather than
  re-registering.
- **Private keys stay `0600`.** The configurable `[store].dir_mode`/`group` touch
  only the store directory and group ownership — never per-file modes. Key-bearing
  artifacts remain `0600` (chgrp on a `0600` file grants the group nothing), and
  archived snapshots preserve each file's original mode, so keys stay locked in the
  archive. `status` reads only public material and never prints the EAB HMAC.
- **New at-rest consideration:** with `archive_keep > 0`, historical private keys are
  retained (`0600`) under the locked store and stay valid until their certs expire —
  keep the value modest and protect store backups. History is off by default.
- **`destroy --keep-account`** narrows an already-confirmed destructive operation; it
  keeps the account and skips system-trust removal.


## [v0.2.0] — 2026-06-16

### Features

- **cli:** `--env-file <path>` loads DNS/CA credentials from a systemd
  EnvironmentFile for a manual run, so you no longer have to export every variable
  by hand. Repeatable; an existing environment variable always wins; secrets are
  never logged.
- **packaging:** the network installer now uninstalls — `curl … | sudo sh -s --
  --uninstall` (add `--purge` to also remove the store, config, and the `syscert`
  user), no clone needed. It delegates to `install.sh`, the single source of truth.
- **web:** a `/healthz` endpoint plus Docker and Traefik healthchecks for the
  website container.

### Fixes

- **security:** use the `#nosec` directive so gosec honours the baseline
  suppressions.
- **ci:** keep `web/scripts` in the Docker build context so the vendor prebuild
  runs.

### Documentation

- **Vault PKI ACME supports `dns-01`** — corrected the stale "Vault has no dns-01"
  claim across the docs and added an annotated `vault-dns-01.toml` example
  (role-scoped directory + EAB).
- Documented `--env-file` across the README, FAQ, configuration reference, quick
  start, and troubleshooting; documented the uninstall paths on the install page
  and in advanced-install.
- Slimmed the README to a quick-start hub with CI status badges.

### Risk & Security

Low risk — no changes to certificate issuance, key handling, or the privilege model.

- **`--purge` is now gated.** The destructive uninstall (store, config, secrets,
  and the `syscert` user) prompts for confirmation on the controlling terminal —
  working even under `curl … | sh`, where stdin is the pipe — and refuses to
  proceed without a terminal unless `SYSCERT_ASSUME_YES=1` is set. This makes an
  already-existing operation safer.
- **`--env-file` does not widen secret exposure.** It is opt-in (nothing is read
  without the flag), loads the same `/etc/syscert/secrets` the systemd unit already
  uses, never overrides a variable already in the environment, and never logs
  values (parse errors cite only the line number).
- **Network uninstall** fetches `packaging/install.sh` over TLS from the pinned tag
  (or `main`) and delegates to it — the same trust model as install; the release
  binary remains checksum-verified.
- **Known issue:** `npm audit` reports 5 high advisories in the website's
  build-time dependencies (Astro toolchain). They affect the build, not the
  published static site or the tool binary, and are tracked for a separate web
  dependency bump.


## [v0.1.0] — 2026-06-14

First feature release with full documentation, a one-line installer, and a
repeatable release process.

### Highlights

- **Adopted [Semantic Versioning](https://semver.org/) + [Conventional Commits](https://www.conventionalcommits.org/).**
  Commit subjects now drive the version bump (`feat`→minor, `fix`→patch,
  breaking→major) and the changelog; `scripts/prerelease.sh` audits readiness and
  `scripts/release.sh` publishes — see
  [RELEASING.md](https://github.com/tfindley/syscert/blob/main/RELEASING.md).

### Features

- **acme:** External Account Binding (EAB) — set `[acme.eab].kid` in the config and
  supply the HMAC via `SYSCERT_EAB_HMAC`, for CAs that require it (Vault `eab_policy`,
  step-ca `requireEAB`, ZeroSSL / Google / SSL.com).
- **web:** a full documentation site — quick start, configuration, sample configs,
  distributing, troubleshooting, FAQ, roadmap, and changelog — single-sourced from
  Markdown in `docs/` and rendered on GitHub and the website.
- **packaging:** a one-line network installer (`net-install.sh`, served at
  `/install.sh`) that downloads the matching release binary, verifies its checksum,
  and delegates to `install.sh`. The site shows the real release version + checksums.

### Fixes

- **packaging:** the installer enables the systemd timer but no longer starts it
  until the config is in place.

### Changed

- **cli:** shared flag parsing across subcommands, with `--flag`-style usage output.
- Docs are now canonical Markdown under `docs/`; the README is a lean overview.

### Continuous integration

- Added `govulncheck` + `gosec` scanning on every push/PR; the site rebuilds when the
  docs, changelog, or installer change, and on each published release.

### Risk & Security

Reviewed with a simplify pass and a security review (no findings). EAB adds an
opt-in account-registration path only: the HMAC is read from the environment,
validated before any network call, and never logged — no change to certificate
validation or the default issuance path. The one-line installer checksum-verifies
the downloaded binary before use. The remaining changes are docs, CI, and tooling.

