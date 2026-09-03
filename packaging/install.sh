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
    log "Writing secrets template ${SECRETS_FILE} (0640)"
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

  grant_distribute_paths

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

  # Before the binary goes: it is what derives the granted directory list.
  revoke_distribute_paths

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

# grant_distribute_paths makes the configured [[distribute]] targets actually
# writable by the sandboxed, non-root service. Two separate barriers, both of
# which produce a confusing errno at renewal time if left in place:
#
#   1. ProtectSystem=strict leaves everything outside ReadWritePaths read-only.
#      The binary derives the directory list from the config (shell must not
#      parse TOML) and writes a drop-in.
#   2. A target directory owned by root, e.g. /etc/cockpit/ws-certs.d at 0755,
#      denies the syscert user outright — creating a file needs write on the
#      directory, and CAP_CHOWN does not grant that. A POSIX ACL grants exactly
#      this one user on this one directory, leaving the owner/group the owning
#      package expects untouched.
#
# On a first install the config is the starter template with no real targets, so
# this is close to a no-op; re-run the installer after editing the config. Never
# fatal: a host without ACL support still installs, it just needs the manual step.
grant_distribute_paths() {
  local dirs d
  [ -e "$CONF_FILE" ] || return 0

  if ! "$BIN_DEST" systemd-paths --write --config "$CONF_FILE" >/dev/null 2>&1; then
    warn "could not write the systemd drop-in — run 'syscert systemd-paths --write' after fixing the config"
    return 0
  fi
  log "Wrote ${UNIT_DIR}/syscert.service.d/10-distribute-paths.conf (sandbox grant)"

  dirs=$("$BIN_DEST" systemd-paths --dirs --config "$CONF_FILE" 2>/dev/null) || return 0
  [ -n "$dirs" ] || return 0

  if ! command -v setfacl >/dev/null 2>&1; then
    warn "setfacl not found — grant ${SVC_USER} write on each distribute target directory yourself:"
    printf '%s\n' "$dirs" | while IFS= read -r d; do
      if [ -n "$d" ] && [ "$d" != "$STORE_DIR" ]; then
        warn "  chgrp ${SVC_GROUP} $d && chmod g+w $d"
      fi
    done
    return 0
  fi

  printf '%s\n' "$dirs" | while IFS= read -r d; do
    # The store is already syscert-owned; only the foreign directories need this.
    if [ -n "$d" ] && [ "$d" != "$STORE_DIR" ]; then
      # A mistyped target such as path = "/etc/x.pem" yields the directory /etc.
      # Granting the service write across a whole top-level tree is never what
      # was meant, so refuse it and make the operator fix the path.
      if ! printf '%s' "$d" | grep -Eq '^(/[^/]+){2,}/?$'; then
        warn "refusing to grant $d — too broad; check the [[distribute]] path in $CONF_FILE"
      elif [ ! -d "$d" ]; then
        warn "$d does not exist yet — create it, then re-run (syscert writes there)"
      elif setfacl -m "u:${SVC_USER}:rwx" "$d" 2>/dev/null; then
        log "Granted ${SVC_USER} write on $d (POSIX ACL)"
      else
        warn "could not set an ACL on $d — grant it manually: chgrp ${SVC_GROUP} $d && chmod g+w $d"
      fi
    fi
  done
  return 0
}

# revoke_distribute_paths undoes grant_distribute_paths on uninstall. The config
# is still readable at this point, so the same directory list can be re-derived.
revoke_distribute_paths() {
  rm -rf "${UNIT_DIR}/syscert.service.d"
  [ -e "$CONF_FILE" ] && [ -x "$BIN_DEST" ] || return 0
  command -v setfacl >/dev/null 2>&1 || return 0

  local dirs d
  dirs=$("$BIN_DEST" systemd-paths --dirs --config "$CONF_FILE" 2>/dev/null) || return 0
  # Every branch must succeed: a directory with no ACL to strip is the normal
  # case, and under `set -e` a bare `setfacl && log` would abort the uninstall.
  printf '%s\n' "$dirs" | while IFS= read -r d; do
    if [ -n "$d" ] && [ "$d" != "$STORE_DIR" ] && [ -d "$d" ]; then
      if setfacl -x "u:${SVC_USER}" "$d" 2>/dev/null; then
        log "Removed the ${SVC_USER} ACL from $d"
      fi
    fi
  done
  return 0
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
# This file is 0640 root:syscert and must never be world-readable.
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
      -h|--help)   sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
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
