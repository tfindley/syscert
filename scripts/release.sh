#!/usr/bin/env bash
#
# Publish a prepared SysCert release: tag + push. CI does the build/publish.
#
#   scripts/release.sh v0.1.0       # the version scripts/prerelease.sh prepared
#   scripts/release.sh --yes v0.1.0
#
# Run scripts/prerelease.sh FIRST — it audits readiness, decides the version, and
# writes the CHANGELOG.md entry this script requires. Pushing the tag triggers
# release.yml (build amd64/arm64 + sha256sums + provenance + GitHub Release);
# once the release is published, web.yml rebuilds the site, which fetches the new
# version + checksums at build time.
#
# The website version is independent (web/package.json → site-vX.Y.Z) and the
# site's displayed tool version is fetched at build, so this script does NOT touch
# the web tree — it only tags the tool release.
#
set -euo pipefail

ASSUME_YES=no
VER=""
for a in "$@"; do
  case "$a" in
    --yes|-y)  ASSUME_YES=yes ;;
    -h|--help) sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*)        echo "unknown flag: $a" >&2; exit 2 ;;
    *)         VER="$a" ;;
  esac
done
[ -n "$VER" ] || { echo "error: pass the version, e.g. scripts/release.sh v0.1.0" >&2; exit 2; }

cd "$(git rev-parse --show-toplevel)"
VER="v${VER#v}"
echo "$VER" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || { echo "error: '$VER' is not vX.Y.Z" >&2; exit 2; }

# preconditions
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || { echo "error: releases are cut from main" >&2; exit 1; }
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "error: working tree is dirty — commit the prerelease prep first" >&2; exit 1
fi
git fetch -q --tags origin || true
if git rev-parse -q --verify origin/main >/dev/null 2>&1; then
  [ "$(git rev-list --count HEAD..origin/main)" = "0" ] || { echo "error: behind origin/main — pull first" >&2; exit 1; }
fi
git rev-parse -q --verify "refs/tags/$VER" >/dev/null 2>&1 && { echo "error: tag $VER already exists" >&2; exit 1; }

# the changelog entry is the contract with prerelease — refuse to publish without it
grep -q "^## \[$VER\]" CHANGELOG.md 2>/dev/null \
  || { echo "error: no CHANGELOG.md section for $VER — run scripts/prerelease.sh first" >&2; exit 1; }

echo "Tag + push $VER → triggers release.yml (build/sign/publish) + web.yml (site rebuild)."
if [ "$ASSUME_YES" != "yes" ]; then
  printf "Proceed? [y/N] "; read -r r
  case "$r" in y|Y|yes) ;; *) echo "aborted"; exit 1 ;; esac
fi

git tag -a "$VER" -m "SysCert $VER"
git push origin main "$VER"

echo
echo "pushed $VER. Watch the build:  gh run watch --exit-status"
echo "release: https://github.com/tfindley/syscert/releases/tag/$VER"
