# Releasing SysCert

Two commands, and one source of truth: the git tag. `prerelease` audits and prepares the
release, `release` publishes it, and CI takes it from there to build.

## TL;DR

```sh
scripts/prerelease.sh                 # audit + propose the next version from commits
#   …fix anything it flags, fill the CHANGELOG "Risk & Security" note, commit the prep…
scripts/prerelease.sh --attest-security   # re-run until READY (attest the security review)
scripts/release.sh v0.1.0             # tag + push the version it prepared
```

## What each step does

`scripts/prerelease.sh [patch|minor|major|X.Y.Z]` runs every readiness gate: build/test/lint,
tool-vs-docs command/flag parity, example-config validation, version-stamp and release-asset
names, `govulncheck`, `npm audit`, systemd unit verify, and an SBOM when `syft` is installed. It
works out the next version from your Conventional Commits (or takes an explicit bump you hand
it), scaffolds the `CHANGELOG.md` section grouped by commit type with a **Risk & Security**
placeholder for you to fill, and writes `RELEASE-READINESS.md` with a PASS/FAIL/WARN verdict. It
never tags or pushes, so run it as often as you like. The security-review gate stays red until
you pass `--attest-security`, and you should only do that once you've actually run
`/security-review` on the diff.

`scripts/release.sh vX.Y.Z` does the final guard check (clean `main`, in sync, the tag is new,
the CHANGELOG entry exists), then creates the annotated tag and pushes it. That push triggers
`release.yml`, which builds amd64/arm64, writes `sha256sums.txt`, generates provenance, and cuts
the GitHub Release. Once that's published, `web.yml` rebuilds the site, pulling the new version
and checksums at build time.

The binary gets its version stamped from the tag (`-ldflags -X main.version`), so the tag, the
release, what `syscert version` prints, and the website all agree.

## Versioning

The tool versions off a `vX.Y.Z` git tag that drives `release.yml`, plain Semantic Versioning.

The website carries its own semver in `web/package.json`, which becomes the image tag
`ghcr.io/<owner>/syscert-web:site-vX.Y.Z` (alongside `latest` and `sha-<short>`). Bump that when
the site changes; it doesn't have to match the tool. The site keeps up with the tool by content
rather than by number, fetching and showing the latest tool release at build time and rebuilding
whenever a `release: published` event fires. The `version` in `web/src/consts.ts` is just the
offline fallback for that fetch, so you don't need to bump it every release.

## Conventional Commits

Commit subjects follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[(scope)][!]: <description>
```

The types are `feat` `fix` `docs` `style` `refactor` `perf` `test` `build` `ci` `chore`
`revert`. A `!` (or a `BREAKING CHANGE:` footer) marks a breaking change.

That subject line drives two things: the version bump (`feat`→minor, `fix`→patch,
breaking→major) and the generated changelog. The check runs through the repo's
[pre-commit](https://pre-commit.com/) framework, reusing `.githooks/commit-msg`. Turn it on once
per clone:

```sh
pre-commit install                          # gitleaks (pre-commit stage)
pre-commit install --hook-type commit-msg   # Conventional Commits check
```
