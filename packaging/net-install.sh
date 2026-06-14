#!/usr/bin/env sh
# ─────────────────────────────────────────────────────────────────────────────
# SysCert one-line network installer.
#
#   curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
#
# It's a security tool piped to root — so read it first. Here is everything it does,
# and nothing else:
#
#   1. Detects your OS (Linux) and CPU arch (amd64 / arm64).
#   2. Resolves the latest SysCert release tag on GitHub.
#   3. Downloads the matching static binary + sha256sums.txt from that release
#      and VERIFIES the checksum (aborts on any mismatch).
#   4. Downloads the systemd packaging (install.sh + units) pinned to the SAME tag.
#   5. Hands off to the standard packaging/install.sh, which creates the `syscert`
#      user, lays down /var/lib/syscert + /etc/syscert, installs the units, and
#      enables (does NOT start) the timer. See ADR-0034: the binary never self-installs.
#
# No telemetry, no background process, nothing left behind but what install.sh writes.
# Everything is fetched to a temp dir that is removed on exit.
#
# Pin a specific version:   SYSCERT_VERSION=v0.1.0 curl -fsSL …/install.sh | sudo sh
# ─────────────────────────────────────────────────────────────────────────────
set -eu

readonly REPO="tfindley/syscert"
readonly GH="https://github.com"
readonly RAW="https://raw.githubusercontent.com"
readonly API="https://api.github.com"

log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

# Re-exec with sudo when invoked without root and a real script file exists. The
# documented one-liner pipes to `sudo sh`, so the common path is already root;
# this just helps `curl … | sh` (no sudo) when run from a saved file.
ensure_root() {
  [ "$(id -u)" -eq 0 ] && return 0
  if [ -f "$0" ] && have sudo; then
    log "Not root — re-executing with sudo"
    exec sudo -- sh "$0" "$@"
  fi
  die "must run as root — use: curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh"
}

detect_platform() {
  os="$(uname -s)"
  [ "$os" = "Linux" ] || die "unsupported OS '$os' — SysCert targets Linux + systemd"
  m="$(uname -m)"
  case "$m" in
    x86_64 | amd64)  ARCH="amd64" ;;
    aarch64 | arm64) ARCH="arm64" ;;
    *) die "unsupported architecture '$m' — prebuilt binaries are amd64 and arm64 only" ;;
  esac
}

check_tools() {
  if   have curl; then DL="curl"
  elif have wget; then DL="wget"
  else die "need curl or wget to download SysCert"
  fi
  if   have sha256sum; then SHA="sha256sum"
  elif have shasum;    then SHA="shasum -a 256"
  else die "need sha256sum or shasum to verify the download"
  fi
  have bash      || die "need bash — the system installer (packaging/install.sh) requires it"
  have systemctl || die "systemctl not found — SysCert targets systemd hosts"
}

# Download a URL to a file ($1 = url, $2 = dest).
dl_to() {
  if [ "$DL" = "curl" ]; then curl -fsSL "$1" -o "$2"
  else wget -qO "$2" "$1"; fi
}

# Download a URL to stdout ($1 = url).
dl_stdout() {
  if [ "$DL" = "curl" ]; then curl -fsSL "$1"
  else wget -qO- "$1"; fi
}

# Resolve the release tag to install into $VERSION.
resolve_version() {
  if [ -n "${SYSCERT_VERSION:-}" ]; then
    VERSION="$SYSCERT_VERSION"
    return
  fi
  VERSION=""
  # Cheap + rate-limit-free: read the tag off the /releases/latest redirect.
  if [ "$DL" = "curl" ]; then
    eff="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$GH/$REPO/releases/latest" 2>/dev/null || true)"
    case "$eff" in
      */releases/tag/*) VERSION="${eff##*/}" ;;
    esac
  fi
  # Fallback: the releases API (works with curl or wget).
  if [ -z "$VERSION" ]; then
    VERSION="$(dl_stdout "$API/repos/$REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  case "$VERSION" in
    v[0-9]*) : ;;
    *) die "could not resolve the latest release — set SYSCERT_VERSION=vX.Y.Z to pin one" ;;
  esac
}

main() {
  ensure_root "$@"
  detect_platform
  check_tools
  resolve_version

  TMP="$(mktemp -d "${TMPDIR:-/tmp}/syscert-install.XXXXXX")"
  trap 'rm -rf "$TMP"' EXIT INT TERM

  bin="syscert-linux-$ARCH"
  base="$GH/$REPO/releases/download/$VERSION"

  log "Installing SysCert $VERSION ($ARCH)"
  dl_to "$base/$bin"            "$TMP/$bin"          || die "failed to download $bin from $VERSION"
  dl_to "$base/sha256sums.txt" "$TMP/sha256sums.txt" || die "failed to download sha256sums.txt"

  log "Verifying checksum"
  (
    cd "$TMP"
    grep "[[:space:]]${bin}\$" sha256sums.txt > sha256.one 2>/dev/null || true
    [ -s sha256.one ] || die "no checksum entry for $bin in sha256sums.txt"
    $SHA -c sha256.one
  ) || die "checksum verification FAILED — refusing to install"
  chmod +x "$TMP/$bin"

  log "Fetching systemd packaging ($VERSION)"
  mkdir -p "$TMP/packaging/systemd"
  praw="$RAW/$REPO/$VERSION/packaging"
  dl_to "$praw/install.sh"               "$TMP/packaging/install.sh"          || die "failed to fetch packaging/install.sh"
  dl_to "$praw/systemd/syscert.service"  "$TMP/packaging/systemd/syscert.service" || die "failed to fetch syscert.service"
  dl_to "$praw/systemd/syscert.timer"    "$TMP/packaging/systemd/syscert.timer"   || die "failed to fetch syscert.timer"

  log "Running the system installer"
  bash "$TMP/packaging/install.sh" "$TMP/$bin"

  printf '\n'
  log "Done. Quick start → https://syscert.tfindley.dev/docs/quick-start/"
}

main "$@"
