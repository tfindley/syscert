#!/usr/bin/env bash
#
# SysCert installer (external to the binary by design — see docs ADR-0034).
# Idempotent: safe to re-run. Requires root.
#
#   sudo packaging/install.sh [PATH_TO_BINARY]   install (binary defaults to ../syscert)
#   sudo packaging/install.sh --uninstall        remove units + binary (keeps data)
#   sudo packaging/install.sh --uninstall --purge also remove /var/lib/syscert, /etc/syscert, user
#   --purge prompts for confirmation on the terminal; SYSCERT_ASSUME_YES=1 skips it.
#
set -euo pipefail

readonly SVC_USER="syscert"
readonly SVC_GROUP="syscert"
readonly STORE_DIR="/var/lib/syscert"
readonly CONF_DIR="/etc/syscert"
readonly CONF_FILE="${CONF_DIR}/syscert.toml"
readonly SECRETS_FILE="${CONF_DIR}/secrets"
readonly DEFAULTS_FILE="/etc/default/syscert"
readonly BIN_DEST="/usr/local/bin/syscert"
readonly UNIT_DIR="/etc/systemd/system"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

require_root() { [ "$(id -u)" -eq 0 ] || die "must run as root (try: sudo $0 $*)"; }

require_systemd() {
  command -v systemctl >/dev/null 2>&1 || die "systemctl not found — this installer targets systemd hosts"
}

selinux_active() { [ -d /sys/fs/selinux ] && command -v restorecon >/dev/null 2>&1; }

# ----------------------------------------------------------------------------

install_syscert() {
  local bin_src="${1:-${SCRIPT_DIR}/../syscert}"
  [ -x "$bin_src" ] || die "binary not found/executable at '$bin_src' — build it first (go build -o syscert ./cmd/syscert) or pass the path"

  log "Creating ${SVC_GROUP} group and ${SVC_USER} system user (if absent)"
  getent group "$SVC_GROUP" >/dev/null 2>&1 || groupadd --system "$SVC_GROUP"
  if ! getent passwd "$SVC_USER" >/dev/null 2>&1; then
    local nologin="/usr/sbin/nologin"
    [ -x "$nologin" ] || nologin="/sbin/nologin"
    useradd --system --gid "$SVC_GROUP" --home-dir "$STORE_DIR" \
      --no-create-home --shell "$nologin" "$SVC_USER"
  fi

  # Default 0700 syscert:syscert; [store].dir_mode / [store].group widen this on each
  # run if set (private keys stay 0600 regardless). See docs/configuration.md.
  log "Creating store ${STORE_DIR} (0700) and config dir ${CONF_DIR}"
  install -d -o "$SVC_USER" -g "$SVC_GROUP" -m 0700 "$STORE_DIR"
  install -d -o root -g root -m 0755 "$CONF_DIR"

  log "Installing binary → ${BIN_DEST}"
  install -o root -g root -m 0755 "$bin_src" "$BIN_DEST"

  if [ ! -e "$CONF_FILE" ]; then
    log "Writing starter config ${CONF_FILE} (edit before first run)"
    write_config_template
  else
    log "Keeping existing ${CONF_FILE}"
  fi

  if [ ! -e "$SECRETS_FILE" ]; then
    log "Writing secrets template ${SECRETS_FILE} (0600)"
    write_secrets_template
  else
    log "Keeping existing ${SECRETS_FILE}"
  fi

  if [ ! -e "$DEFAULTS_FILE" ]; then
    log "Writing defaults template ${DEFAULTS_FILE} (operator settings; optional)"
    write_defaults_template
  else
    log "Keeping existing ${DEFAULTS_FILE}"
  fi

  log "Installing systemd units → ${UNIT_DIR}"
  install -o root -g root -m 0644 "${SCRIPT_DIR}/systemd/syscert.service" "${UNIT_DIR}/syscert.service"
  install -o root -g root -m 0644 "${SCRIPT_DIR}/systemd/syscert.timer"   "${UNIT_DIR}/syscert.timer"

  if selinux_active; then
    # Relabel the binary too: installed from /root it keeps admin_home_t, which
    # systemd (init_t) cannot execute — restorecon gives it bin_t so the timer
    # can run it without a permissive policy module. Idempotent.
    log "SELinux active — relabeling ${BIN_DEST}, ${STORE_DIR}, and ${CONF_DIR}"
    restorecon -R "$BIN_DEST" "$STORE_DIR" "$CONF_DIR" || warn "restorecon failed (continuing)"
  fi

  # Enable (not --now): don't run against the unconfigured starter config. The
  # operator starts the timer after editing the config (step 4 below).
  log "Enabling syscert.timer (not started — start it after configuring)"
  systemctl daemon-reload
  systemctl enable syscert.timer

  cat <<EOF

SysCert installed. The timer is enabled but NOT started yet. Next steps:
  1. Edit ${CONF_FILE}        (subject, CA, challenge, distribute targets)
  2. Add credentials to ${SECRETS_FILE}   (e.g. GANDIV5_PERSONAL_ACCESS_TOKEN=...)
  3. Test once:   sudo -u ${SVC_USER} ${BIN_DEST} --config ${CONF_FILE} --staging
  4. Start it:    sudo systemctl start syscert.timer
  Timer status:   systemctl list-timers syscert.timer
EOF
}

# assume_yes reports whether SYSCERT_ASSUME_YES authorises a non-interactive purge.
assume_yes() {
  case "${SYSCERT_ASSUME_YES:-}" in
    1 | y | Y | yes | YES | true | TRUE) return 0 ;;
    *) return 1 ;;
  esac
}

# confirm_purge requires explicit confirmation before the irreversible purge. It
# reads from the controlling terminal (/dev/tty), so it works even when stdin is a
# pipe (curl ... | sh). SYSCERT_ASSUME_YES=1 skips the prompt; with no terminal and
# no override it aborts rather than guess. Call this BEFORE removing anything.
confirm_purge() {
  assume_yes && return 0
  # Open the controlling terminal on fd 3. The group keeps the 2>/dev/null from
  # both leaking the open error and permanently clobbering the shell's stderr.
  if ! { exec 3<>/dev/tty; } 2>/dev/null; then
    die "--purge permanently deletes ${STORE_DIR} (keys + certs), ${CONF_DIR} (config + secrets), and the ${SVC_USER} user, but there is no terminal to confirm on — re-run with SYSCERT_ASSUME_YES=1 to proceed non-interactively"
  fi
  {
    printf '\033[1;33mWARNING:\033[0m --purge will PERMANENTLY delete:\n'
    printf '    %s   (private keys + certificates)\n' "$STORE_DIR"
    printf '    %s        (config + secrets)\n' "$CONF_DIR"
    printf '    the %s system user and group\n' "$SVC_USER"
    printf "Type 'yes' to continue: "
  } >&3
  local reply=""
  IFS= read -r reply <&3 || die "could not read confirmation"
  exec 3>&-
  [ "$reply" = "yes" ] || die "aborted — nothing was changed"
}

uninstall_syscert() {
  local purge="${1:-no}"
  if [ "$purge" = "purge" ]; then
    confirm_purge
  fi
  log "Disabling syscert.timer"
  systemctl disable --now syscert.timer 2>/dev/null || true

  log "Removing units + binary"
  rm -f "${UNIT_DIR}/syscert.timer" "${UNIT_DIR}/syscert.service" "$BIN_DEST"
  systemctl daemon-reload

  if [ "$purge" = "purge" ]; then
    log "Purging data, config, and the ${SVC_USER} user"
    rm -rf "$STORE_DIR" "$CONF_DIR"
    rm -f "$DEFAULTS_FILE"
    userdel "$SVC_USER" 2>/dev/null || true
    groupdel "$SVC_GROUP" 2>/dev/null || true
  else
    log "Kept ${STORE_DIR}, ${CONF_DIR}, and ${DEFAULTS_FILE} (use --purge to remove)"
  fi
  log "Uninstalled."
}

write_config_template() {
  cat > "$CONF_FILE" <<'EOF'
# SysCert configuration. See https://github.com/tfindley/syscert for the full reference.
[cert]
# hostname = "host.example.com"   # defaults to the system FQDN; errors if none
key_type = "ec256"

[acme]
ca        = "letsencrypt"          # letsencrypt | custom (custom = Vault / step-ca / any internal ACME CA)
email     = "you@example.com"
challenge = "dns-01"
# directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"   # required when ca = "custom"

[acme.dns]
provider = "gandiv5"               # any lego DNS provider; creds go in the secrets file
# propagation_check = "authoritative"   # all (default) | authoritative | off — use "authoritative"
                                        # if the host's resolver is split-horizon/VPN/slow

[store]
path = "/var/lib/syscert"

# [[distribute]]
# artifact = "fullchain"
# path     = "/etc/nginx/tls/fullchain.pem"
# owner    = "root"
# group    = "root"
# mode     = "0644"
EOF
  # 0640 root:syscert, not world-readable: the config carries the internal
  # directory_url, ACME email, and EAB kid. The unit runs as User=syscert
  # (group syscert), so the service retains read. Mirrors the secrets file.
  chown root:"$SVC_GROUP" "$CONF_FILE"
  chmod 0640 "$CONF_FILE"
}

write_secrets_template() {
  umask 077
  cat > "$SECRETS_FILE" <<'EOF'
# DNS provider / CA credentials, sourced by the systemd unit (EnvironmentFile).
# This file is 0600 and must never be world-readable.
#
# Set the variables your DNS provider needs (one KEY=value per line). Find the
# exact names for your provider at: https://go-acme.github.io/lego/dns/
# Examples:
# GANDIV5_PERSONAL_ACCESS_TOKEN=replace-me
# CLOUDFLARE_DNS_API_TOKEN=replace-me
#
# External Account Binding HMAC key (base64url), if your CA requires EAB and you
# set [acme.eab] kid in the config:
# SYSCERT_EAB_HMAC=replace-me
EOF
  chown root:"$SVC_GROUP" "$SECRETS_FILE"
  chmod 0640 "$SECRETS_FILE"
}

write_defaults_template() {
  install -d -o root -g root -m 0755 "$(dirname "$DEFAULTS_FILE")"
  cat > "$DEFAULTS_FILE" <<'EOF'
# SysCert operator settings, sourced by the systemd unit (EnvironmentFile).
# Non-secret only — DNS/CA credentials belong in /etc/syscert/secrets.
#
# Point SysCert at a non-default config file. Without this, it uses the
# built-in default /etc/syscert/syscert.toml.
# SYSCERT_CONFIG=/etc/syscert/syscert.toml
EOF
  chmod 0644 "$DEFAULTS_FILE"
}

# ----------------------------------------------------------------------------

main() {
  local action="install" binary="" purge="no"
  for arg in "$@"; do
    case "$arg" in
      --uninstall) action="uninstall" ;;
      --purge)     purge="purge" ;;
      -h|--help)   sed -n '2,12p' "$0"; exit 0 ;;
      -*)          die "unknown flag: $arg" ;;
      *)           binary="$arg" ;;
    esac
  done

  require_root "$@"
  require_systemd

  case "$action" in
    install)   install_syscert "$binary" ;;
    uninstall) uninstall_syscert "$purge" ;;
  esac
}

main "$@"
