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

