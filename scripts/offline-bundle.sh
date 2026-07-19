#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Build a self-contained, checksum-verified SysCert install bundle for
# air-gapped / offline hosts.
#
# Run this on a machine that HAS internet (and this repo checked out). It:
#   1. Downloads the release binary + sha256sums.txt for the chosen version/arch.
#   2. Verifies the checksum (and the SLSA provenance, if `gh` is available).
#   3. Fetches packaging/install.sh + the systemd units, pinned to the same tag.
#   4. Bundles them with a small install-offline.sh + README into a .tar.gz.
#
# Carry the tarball to the air-gapped host, verify it against the SHA-256 this
# script prints, unpack, and run `sudo ./install-offline.sh`. See
# docs/advanced-install/offline.md.
#
# Usage:
#   scripts/offline-bundle.sh [--version vX.Y.Z] [--arch amd64|arm64|all] [--output DIR]
#
#   --version   release tag to bundle (default: latest release)
#   --arch      target CPU arch: amd64, arm64, or all (default: amd64)
#   --output    directory to write the tarball(s) into (default: current dir)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

readonly REPO="tfindley/syscert"
readonly GH="https://github.com"
readonly RAW="https://raw.githubusercontent.com"
readonly API="https://api.github.com"

VERSION=""
ARCHES=("amd64")
OUTDIR="$PWD"

log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

usage() {
  sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version|-v) VERSION="${2:-}"; shift 2 || die "--version needs a value" ;;
      --arch|-a)
        case "${2:-}" in
          amd64|arm64) ARCHES=("$2") ;;
          all)         ARCHES=("amd64" "arm64") ;;
          *)           die "--arch must be amd64, arm64, or all" ;;
        esac
        shift 2 ;;
      --output|-o)  OUTDIR="${2:-}"; shift 2 || die "--output needs a value" ;;
      -h|--help)    usage 0 ;;
      *)            die "unknown argument '$1' (try --help)" ;;
    esac
  done
  [ -d "$OUTDIR" ] || die "output directory '$OUTDIR' does not exist"
}

check_tools() {
  if   have curl; then DL="curl"
  elif have wget; then DL="wget"
  else die "need curl or wget"
  fi
  if   have sha256sum; then SHA="sha256sum"
  elif have shasum;    then SHA="shasum -a 256"
  else die "need sha256sum or shasum"
  fi
  have tar || die "need tar"
}

# dl_to URL DEST — download a URL to a file.
dl_to() {
  if [ "$DL" = "curl" ]; then curl -fsSL "$1" -o "$2"
  else wget -qO "$2" "$1"; fi
}
# dl_stdout URL — download a URL to stdout.
dl_stdout() {
  if [ "$DL" = "curl" ]; then curl -fsSL "$1"
  else wget -qO- "$1"; fi
}

resolve_version() {
  [ -n "$VERSION" ] && return 0
  # Cheap + rate-limit-free: read the tag off the /releases/latest redirect.
  if [ "$DL" = "curl" ]; then
    local eff
    eff="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$GH/$REPO/releases/latest" 2>/dev/null || true)"
    case "$eff" in */releases/tag/*) VERSION="${eff##*/}" ;; esac
  fi
  if [ -z "$VERSION" ]; then
    VERSION="$(dl_stdout "$API/repos/$REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  case "$VERSION" in
    v[0-9]*) : ;;
    *) die "could not resolve the latest release — pass --version vX.Y.Z" ;;
  esac
}

# verify_checksum DIR BIN — verify BIN in DIR against DIR/sha256sums.txt.
verify_checksum() {
  local dir="$1" bin="$2"
  (
    cd "$dir"
    grep "[[:space:]]${bin}\$" sha256sums.txt > .sha.one 2>/dev/null || true
    [ -s .sha.one ] || die "no checksum entry for $bin in sha256sums.txt"
    $SHA -c .sha.one
    rm -f .sha.one
  ) || die "checksum verification FAILED for $bin — refusing to bundle"
}

# The static installer shipped inside every bundle. It finds the single bundled
# binary, checks the host arch matches, re-verifies the checksum, then hands off
# to packaging/install.sh. Kept POSIX sh so it runs on a minimal target.
write_installer() {
  cat > "$1" <<'OFFLINE_INSTALLER'
#!/usr/bin/env sh
# SysCert offline installer — verify the bundled binary, then install it.
# Run: sudo ./install-offline.sh    (to remove: sudo packaging/install.sh --uninstall)
set -eu

have() { command -v "$1" >/dev/null 2>&1; }
log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

unset CDPATH
cd -- "$(dirname -- "$0")" || die "cannot enter bundle directory"

# Exactly one syscert-linux-* binary should sit next to this script.
bin=""
for f in syscert-linux-*; do
  [ -f "$f" ] || continue
  [ -z "$bin" ] || die "multiple syscert-linux-* binaries in bundle"
  bin="$f"
done
[ -n "$bin" ] || die "no syscert-linux-* binary found next to this script"

# Host CPU must match the bundle's binary (override with SYSCERT_FORCE_ARCH=1).
case "$bin" in
  *-amd64) want="x86_64 amd64" ;;
  *-arm64) want="aarch64 arm64" ;;
  *)       die "unrecognised binary name '$bin'" ;;
esac
m="$(uname -m)"
case " $want " in
  *" $m "*) : ;;
  *) [ "${SYSCERT_FORCE_ARCH:-}" = "1" ] \
       || die "bundle is $bin but host arch is $m — wrong bundle (SYSCERT_FORCE_ARCH=1 overrides)" ;;
esac

if   have sha256sum; then SHA="sha256sum"
elif have shasum;    then SHA="shasum -a 256"
else die "need sha256sum or shasum to verify the binary"
fi

log "Verifying $bin against sha256sums.txt"
grep "[[:space:]]${bin}\$" sha256sums.txt > .sha.one 2>/dev/null || true
[ -s .sha.one ] || { rm -f .sha.one; die "no checksum entry for $bin"; }
$SHA -c .sha.one || { rm -f .sha.one; die "checksum verification FAILED — refusing to install"; }
rm -f .sha.one

if [ "$(id -u)" -ne 0 ]; then
  have sudo || die "must run as root"
  log "Not root — re-executing with sudo"
  exec sudo -- sh "$0" "$@"
fi

have bash || die "need bash — packaging/install.sh requires it"
[ -x packaging/install.sh ] || die "packaging/install.sh missing from bundle"

log "Installing via packaging/install.sh"
exec bash packaging/install.sh "./$bin"
OFFLINE_INSTALLER
}

write_readme() {
  local name="$1" arch="$2"
  cat > "$name/README.txt" <<EOF
SysCert offline install bundle
==============================

  Version : $VERSION
  Arch    : linux/$arch
  Contents: syscert-linux-$arch, sha256sums.txt, packaging/ (installer + units),
            install-offline.sh, README.txt

Install (on the air-gapped host):

  1. Verify this bundle against the SHA-256 you were given out of band:
       sha256sum syscert-$VERSION-linux-$arch-offline.tar.gz
  2. Unpack and install:
       tar xzf syscert-$VERSION-linux-$arch-offline.tar.gz
       cd syscert-$VERSION-linux-$arch
       sudo ./install-offline.sh

install-offline.sh re-verifies the binary's checksum and refuses a wrong-arch or
tampered binary before installing. It creates the syscert user, lays down
/var/lib/syscert and /etc/syscert, installs the systemd units, and enables (does
NOT start) the timer.

Next: edit /etc/syscert/syscert.toml for your internal CA (directory_url), provide
secrets via env or a 0640 file, then test once and start the timer. Full guide:
https://syscert.tfindley.dev/docs/advanced-install/offline/

Remove: sudo packaging/install.sh --uninstall   (add --purge to remove data too)
EOF
}

build_bundle() {
  local arch="$1"
  local bin="syscert-linux-$arch"
  local name="syscert-$VERSION-linux-$arch"
  local base="$GH/$REPO/releases/download/$VERSION"
  local praw="$RAW/$REPO/$VERSION/packaging"

  local work stage
  work="$(mktemp -d "${TMPDIR:-/tmp}/syscert-bundle.XXXXXX")"
  # shellcheck disable=SC2064
  trap "rm -rf '$work'" RETURN
  stage="$work/$name"
  mkdir -p "$stage/packaging/systemd"

  log "[$arch] downloading $bin + sha256sums.txt ($VERSION)"
  dl_to "$base/$bin"            "$stage/$bin"            || die "failed to download $bin — is $VERSION released for $arch?"
  dl_to "$base/sha256sums.txt"  "$stage/sha256sums.txt"  || die "failed to download sha256sums.txt"

  log "[$arch] verifying checksum"
  verify_checksum "$stage" "$bin"

  if have gh; then
    log "[$arch] verifying build provenance (gh attestation verify)"
    if gh attestation verify "$stage/$bin" --repo "$REPO" >/dev/null 2>&1; then
      log "[$arch] provenance OK"
    else
      warn "[$arch] provenance check did not pass (gh unauthenticated or offline?) — checksum still verified"
    fi
  else
    warn "[$arch] gh not installed — skipping provenance check (checksum still verified)"
  fi

  log "[$arch] fetching packaging pinned to $VERSION"
  dl_to "$praw/install.sh"               "$stage/packaging/install.sh"          || die "failed to fetch packaging/install.sh"
  dl_to "$praw/systemd/syscert.service"  "$stage/packaging/systemd/syscert.service" || die "failed to fetch syscert.service"
  dl_to "$praw/systemd/syscert.timer"    "$stage/packaging/systemd/syscert.timer"   || die "failed to fetch syscert.timer"

  chmod +x "$stage/$bin" "$stage/packaging/install.sh"
  write_installer "$stage/install-offline.sh"
  chmod +x "$stage/install-offline.sh"
  write_readme "$stage" "$arch"

  local out="$OUTDIR/$name-offline.tar.gz"
  tar czf "$out" -C "$work" "$name"

  local sum
  sum="$($SHA "$out" | awk '{print $1}')"
  log "[$arch] wrote $out"
  printf '    sha256: %s\n' "$sum"
}

main() {
  parse_args "$@"
  check_tools
  resolve_version
  log "Building offline bundle(s) for SysCert $VERSION → ${ARCHES[*]}"
  for arch in "${ARCHES[@]}"; do
    build_bundle "$arch"
  done
  log "Done. Carry the tarball + its printed sha256 to the air-gapped host, then run install-offline.sh."
}

main "$@"
