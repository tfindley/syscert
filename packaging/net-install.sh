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
# Uninstall (no clone needed):
#   curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall
#   …add --purge to also remove /var/lib/syscert, /etc/syscert, and the user.
#   --purge asks you to confirm on the terminal; SYSCERT_ASSUME_YES=1 skips it.
#
# Pin a specific version:   SYSCERT_VERSION=v0.1.0 curl -fsSL …/install.sh | sudo sh
# ─────────────────────────────────────────────────────────────────────────────
set -eu

readonly REPO="tfindley/syscert"
readonly GH="https://github.com"
readonly RAW="https://raw.githubusercontent.com"
readonly API="https://api.github.com"

ACTION="install"
PURGE=""

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

check_download_tool() {
  if   have curl; then DL="curl"
  elif have wget; then DL="wget"
  else die "need curl or wget to download SysCert"
  fi
}

check_checksum_tool() {
  if   have sha256sum; then SHA="sha256sum"
  elif have shasum;    then SHA="shasum -a 256"
  else die "need sha256sum or shasum to verify the download"
  fi
}

require_runtime() {
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

do_install() {
  detect_platform
  check_download_tool
  check_checksum_tool
  require_runtime
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

# do_uninstall delegates to the same packaging/install.sh the install path uses,
# so there's a single source of truth for what gets removed (and for the --purge
# confirmation). The uninstall logic is version-independent, so the installer is
# fetched from SYSCERT_VERSION when pinned, otherwise from main.
do_uninstall() {
  check_download_tool
  require_runtime

  ref="${SYSCERT_VERSION:-main}"
  TMP="$(mktemp -d "${TMPDIR:-/tmp}/syscert-uninstall.XXXXXX")"
  trap 'rm -rf "$TMP"' EXIT INT TERM

  log "Fetching the system uninstaller ($ref)"
  dl_to "$RAW/$REPO/$ref/packaging/install.sh" "$TMP/install.sh" \
    || die "failed to fetch packaging/install.sh from $ref"

  log "Running the system uninstaller${PURGE:+ (with --purge)}"
  bash "$TMP/install.sh" --uninstall $PURGE
}

usage() {
  cat <<'EOF'
SysCert network installer / uninstaller.

  ... | sudo sh                               install (latest release)
  ... | sudo sh -s -- --uninstall             remove units + binary (keep data)
  ... | sudo sh -s -- --uninstall --purge     also remove data, config, and the user

Env: SYSCERT_VERSION=vX.Y.Z pins the release; SYSCERT_ASSUME_YES=1 skips the
--purge confirmation.
EOF
}

parse_args() {
  for arg in "$@"; do
    case "$arg" in
      --uninstall) ACTION="uninstall" ;;
      --purge)     PURGE="--purge" ;;
      -h | --help) usage; exit 0 ;;
      *)           die "unknown argument '$arg' — usage: [--uninstall [--purge]]" ;;
    esac
  done
  if [ -n "$PURGE" ] && [ "$ACTION" != "uninstall" ]; then
    die "--purge requires --uninstall"
  fi
}

main() {
  parse_args "$@"
  ensure_root "$@" # single root-check seam; may re-exec via sudo, forwarding args
  if [ "$ACTION" = "uninstall" ]; then
    do_uninstall
  else
    do_install
  fi
}

main "$@"
