## [v0.5.0] — 2026-09-03

### Features

feat(observe): optional Prometheus textfile and Ansible facts outputs (27f4d96)
feat(ansible): finalise the syscert role and gate it in CI (826b132)
feat(packaging): offline install bundle tool + air-gapped install guide (112e624)

### Bug Fixes

fix: reconcile binary, docs, site, role and scripts after a full audit (dab816c)
fix(install): refuse to ACL a top-level directory (3614de7)
fix(distribute): make privileged target directories work (#14) (728380c)
fix(systemd): ReadWritePaths must name each distribute target's directory (1e0101a)
fix(web): tmpfs mounts for the rootless container + smoke test before publish (629c23c)

### Documentation

docs(site): catch the marketing site up with what actually shipped (119f745)
docs: ship the Ansible role — new page, roadmap, compliance scope (e481b01)
docs(configuration): mark reuse_key as accepted but not yet applied (8d314e8)
docs(roadmap): verify, observability, ARI — differentiation vs lego CLI (863c2c7)
docs(comparison): compare SysCert with the lego CLI (befdaf7)
docs: unwrap hard line-wrapping repo-wide (soft-wrap; no content change) (#10) (cc2ae34)
docs: unwrap hard line-wrapping in the new risk + offline pages (c30dbde)
docs(compliance): human-rewrite the risk assessment (voice only) (ca417eb)
docs(compliance): add impartial adoption + operational risk assessment (b48bdf0)

### Risk & Security

Low risk, with one deliberate expansion of the **install-time** footprint. The binary's own posture is unchanged: this release adds no code to key handling, secret sourcing, trust, or store permissions (`git diff v0.4.0..HEAD -- internal/acme internal/store internal/trust internal/envfile` is empty). A security review of `v0.4.0..HEAD` found no blocking issues; the advisory items are recorded in the [security assessment](/docs/compliance/security/).

- **The installer now changes permissions on directories it does not own.** To fix #14, `install.sh` and the Ansible role apply a POSIX ACL (`setfacl -m u:syscert:rwx`) to each configured `[[distribute]]` and `[observe]` directory, and install a systemd drop-in naming them in `ReadWritePaths`. This is the smallest grant that works — one user, one directory, owner and group untouched — chosen over `CAP_DAC_OVERRIDE` or relaxing `ProtectSystem=strict`, either of which would be permanent and host-wide (ADR-0048). Be clear about what it gives away: `rwx` on a directory lets the syscert user create, rename and **delete every file in it**, not only its own artefacts. All three routes now refuse anything shallower than two path components — the binary that writes the drop-in gained that guard in this release, so a mistyped `path = "/etc/x.pem"` can no longer widen the sandbox to all of `/etc` — but two components is a typo guard, not a security boundary: `/etc/sudoers.d` and `/usr/bin` both satisfy it. **Treat your distribute paths as privileged configuration.** Grants are logged per directory and reversed on uninstall, though revocation is derived from the *current* config, so remove a target by re-running the installer rather than by editing the config alone.
- **`[observe]` writes metadata, never secrets.** The new Prometheus textfile and Ansible facts outputs are off by default, write-only, and never read back, so they cannot influence behaviour. Both are `0644` deliberately — node_exporter and Ansible are not the syscert user. The snapshot carries subject, CA *name*, challenge, issuer, serial, validity and renewal times, key type and target presence; no credential, EAB HMAC, directory URL or key material appears in it, verified by running with canary secrets in the environment and grepping both outputs. Label values escape backslash, quote, CR and newline, so a crafted issuer CN cannot inject a metric line.
- **Distribution now attempts every target instead of stopping at the first failure.** One ungranted directory no longer denies every other consumer its renewed certificate. Nothing is written more permissively as a result — a failing target is skipped, never forced — and the run still exits non-zero, so a broken target stays loud.
- **Go 1.26.6 clears nine advisories.** Seven standard-library, plus GO-2026-6061 (grpc → v1.82.1) and GO-2026-5970 (`x/text` → v0.39.0). `govulncheck` reports zero affecting. Direct dependencies remain two.
- **The Ansible role is in scope of the published assessment for the first time.** It verifies the release binary's checksum with no opt-out, renders secrets `0640 root:syscert` under `no_log`, and derives `ReadWritePaths` from the declared targets. Note the limit: the binary and its `sha256sums.txt` come from the same origin, so verification proves integrity, not provenance. `syscert_install_method: local` is the stronger route for high-assurance fleets.
- **`npm audit` reports 7 high advisories in the website's build toolchain** (postcss, sharp/libvips, svgo). Build-time only: the published container is nginx serving pre-rendered static files with no Node runtime, and every input those tools process is first-party. Unrelated to the tool binary. The site container is now fully rootless — nginx as uid 101 on :8080, read-only rootfs, `cap_drop: ALL`, `no-new-privileges`.


## [v0.4.0] — 2026-07-19

### Features

feat(cli): add --interval scheduler mode for non-systemd contexts (7a19f00)
feat(web): build a procedures download zip in CI; slot Procedures in the menu (43f6a19)

### Bug Fixes

fix(ci): exclude web/site/ci-scoped commits from the binary version bump (ac667de)
fix(ci): backward-parity gate falsely flagged real multi-word commands (6194e77)
fix(acme): drop unbuildable dns-persist-01; reject it at config validation (93aa669)
fix(web): serve CSP frame-ancestors as an HTTP header (48439c4)
fix(web): self-host fonts and add a Content-Security-Policy (75673d9)

### Documentation

docs: add Comparison page (syscert vs certbot; when to use which) (#4) (a34adb7)
docs: human-rewrite all public docs + site copy (voice only) (010ca37)
docs: add Comparison page (syscert vs certbot; when to use which) (a8c4138)
docs: add Containerisation section + container reference examples (d9085b3)
docs(spec): containerising syscert — design (5f336c0)
docs: roadmap — add "reissue on config drift" to Next (69e3821)
docs: fix accuracy issues found in the documentation audit (c1fb2e6)
docs: add formal Procedures section (index + 11 SOPs) (0384883)
docs: note self-hosted fonts and CSP in tech-stack page (883f80c)
docs: add an upgrading guide under advanced install (b9416aa)
docs(compliance): add Tech stack and AI-assisted development pages (d0edbef)
docs(compliance): nest the security assessment under a Compliance section (6700cc7)
docs(security): publish the security assessment + risk register (0912953)

### Other

Docs comparison (#6) (912efdf)
ci+pkg: backward command↔docs parity gate; correct secrets log string (87438bd)
Merge branch 'web-surface-headers' (9426efd)

### Risk & Security

Low risk. The only new runtime surface is the opt-in `--interval` scheduler; certificate
issuance, key handling, the privilege model, and secret/trust handling are otherwise
unchanged. A `/security-review` of `v0.3.1..HEAD` found no vulnerabilities.

- **Go 1.26.5 is the headline fix.** It patches two reachable standard-library advisories
  that `govulncheck` flagged on the previous toolchain — GO-2026-5856 (`crypto/tls`) and
  GO-2026-4970 (`os`). The release build now reports zero affecting vulnerabilities.
- **`--interval` is a scheduler, not a daemon.** It's opt-in (bare one-shot stays the
  default), runs no external commands, and reads only a flag/env duration, floored at 1m so
  it can't hammer the CA. `SIGTERM`/`SIGINT` cancel at a cycle boundary, so an in-flight
  issuance or key/store write always finishes before exit — no partial or corrupt key
  material. A failed cycle logs at error level and retries on the next tick; a bad config
  still exits non-zero before the loop even starts. ADR-0046 scopes it as a
  container/appliance scheduler, not a host service — the systemd timer path is unchanged
  (ADR-0033).
- **Dropping `dns-persist-01` only tightens validation.** The unbuildable challenge is now
  rejected up front at `dry-run` with a clear message instead of failing mid-issuance; no
  code path is widened.
- **Nothing changed in secrets, keys, permissions, or trust.** Secrets stay env- or
  `0640`-file-only and unlogged, private keys `0600`, and the internal-CA `trust` commands
  are untouched.
- **Supply chain unchanged.** The release is a CGO-free static binary published with sha256
  checksums and SLSA build provenance. `npm audit` reports one high advisory in the website's
  build-time toolchain (adm-zip, used only to *write* the procedures zip from our own docs) —
  it affects the site build, not the tool binary, and is tracked for a separate web
  dependency bump.


## [v0.3.1] — 2026-06-18

### Fixes

- **security (RHEL):** `install.sh` now relabels the installed binary to `bin_t` so
  systemd can execute it on an SELinux-enforcing host — no permissive policy module
  needed. (#1)
- **security (RHEL):** the write commands (`ensure`/`issue`/`renew`/`void`/`distribute`)
  refuse early, with an actionable message, when the store can't be safely written: as
  root over a `syscert`-owned store (which would create root-owned files the timer can't
  renew, #2), or as a user who doesn't own the store (replacing the raw
  `mkdir … permission denied`, #3). Read-only commands are unaffected.
- **security (RHEL):** the starter `syscert.toml` is installed `0640 root:syscert` (was
  world-readable `0644`), so the internal `directory_url`, ACME email, and EAB `kid`
  aren't readable by every local user. (#3)
- **web:** the site bakes the correct tool version again. The version was resolved inside
  a buildx-cached Docker layer, so a rebuild on an unchanged source commit reused a stale
  layer and kept showing the previous release. It's now passed as a `SITE_VERSION`
  build-arg (deterministic + cache-busting), and the site auto-rebuilds after a release
  via a `workflow_run` chain (the `release: published` event never fired for
  `GITHUB_TOKEN`-created releases).
- **web:** docs menu reordered; **Advanced install** split into *Manually* / *Compile from
  source* / *As a cron job* (for appliances without systemd, e.g. Asustor) sub-pages; the
  sidebar submenu now indents correctly under its parent.

### Risk & Security

Low risk — bug fixes and hardening only; no change to certificate issuance, key handling,
or the privilege model's defaults. A `/security-review` of `v0.3.0..HEAD` found no
vulnerabilities.

- **The store-ownership preflight is purely restrictive** — it can only refuse a write
  (exit 1), never proceed where it previously wouldn't, and never elevates privilege. It
  reads only the store path, owner uid, and username — no secret material.
- **Permissions only tighten.** `syscert.toml` moves `0644 → 0640`; secrets stay `0640`,
  private keys `0600`, and the EAB HMAC is still never logged or printed.
- **SELinux:** the installer relabels the binary to `bin_t` (a label re-derived from
  policy, not an arbitrary grant); no permissive module is shipped.
- **Behaviour to note:** running a write command as a user that doesn't own the store —
  including bare `sudo syscert` (root) over a `syscert`-owned store — now refuses with
  guidance instead of silently mis-owning files. Run as the store owner
  (`sudo -u syscert syscert …`); the systemd timer already does.


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

