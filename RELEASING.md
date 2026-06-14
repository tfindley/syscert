# Releasing SysCert

Two commands, one source of truth (the git tag). `prerelease` audits and prepares;
`release` publishes; CI builds.

## TL;DR

```sh
scripts/prerelease.sh                 # audit + propose the next version from commits
#   …fix anything it flags, fill the CHANGELOG "Risk & Security" note, commit the prep…
scripts/prerelease.sh --attest-security   # re-run until READY (attest the security review)
scripts/release.sh v0.1.0             # tag + push the version it prepared
```

## What each step does

- **`scripts/prerelease.sh [patch|minor|major|X.Y.Z]`** — runs every readiness gate
  (build/test/lint, tool-vs-docs command/flag parity, example-config validation,
  version-stamp + release-asset names, `govulncheck`, `npm audit`, systemd unit
  verify, SBOM if `syft` is present), **infers the next version from Conventional
  Commits** (or takes an explicit bump), scaffolds the `CHANGELOG.md` section
  (grouped by commit type) with a **Risk & Security** placeholder you must fill,
  and writes `RELEASE-READINESS.md` with a PASS/FAIL/WARN verdict. It never tags
  or pushes — safe to run repeatedly. The security-review gate only passes when you
  pass `--attest-security` (after actually running `/security-review` on the diff).
- **`scripts/release.sh vX.Y.Z`** — final guard (clean `main`, in sync, tag is new,
  the CHANGELOG entry exists), then creates the annotated tag and pushes. That tag
  triggers `release.yml` (build amd64/arm64 → `sha256sums.txt` → provenance →
  GitHub Release); once published, `web.yml` rebuilds the site, which fetches the
  new version + checksums at build time.

The binary's version is stamped from the tag (`-ldflags -X main.version`), so the
tag, the release, the binary's `syscert version`, and the website all line up.

## Versioning

- **Tool:** `vX.Y.Z` git tag → release.yml. Semantic Versioning.
- **Website:** independent semver in `web/package.json` → image tag
  `ghcr.io/<owner>/syscert-web:site-vX.Y.Z` (+ `latest`, `sha-<short>`). Bump it for
  site changes; it does not need to match the tool. The site stays "in line" with
  the tool by **content** — it fetches and displays the latest tool release at
  build time, and rebuilds on `release: published`.
- `web/src/consts.ts` `version` is only the **offline fallback** for that fetch; it
  doesn't need bumping per release.

## Conventional Commits

Commit subjects follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[(scope)][!]: <description>
```

Types: `feat` `fix` `docs` `style` `refactor` `perf` `test` `build` `ci` `chore`
`revert`. A `!` (or a `BREAKING CHANGE:` footer) marks a breaking change.

This drives both the version bump (`feat`→minor, `fix`→patch, breaking→major) and
the generated changelog. Enable the local check once:

```sh
git config core.hooksPath .githooks
```
