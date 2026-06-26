#!/bin/sh
# reload-helper.sh — OPTIONAL / ADVANCED
# ------------------------------------------------------------------------------
# Watches the syscert store with inotifywait and runs a reload command when a
# new cert is written. For apps that lack a native cert-watcher.
#
# Prefer app-native reload where available:
#   nginx:   `nginx -s reload` via cron, or a periodic SIGHUP
#   Traefik: watches its configured cert dir automatically (no helper needed)
#   Caddy:   watches and reloads automatically (no helper needed)
#
# This script is Linux-only (inotifywait). Install: apk add inotify-tools
#
# Usage (as a Docker Compose service or a background process):
#   ./reload-helper.sh /var/lib/syscert "docker exec nginx nginx -s reload"
#
# Arguments:
#   $1  — directory to watch (the syscert store, e.g. /var/lib/syscert)
#   $2  — reload command (quoted if it contains spaces)
# ------------------------------------------------------------------------------

WATCH_DIR="${1:-/var/lib/syscert}"
RELOAD_CMD="${2:-nginx -s reload}"

if ! command -v inotifywait >/dev/null 2>&1; then
  echo "reload-helper: inotifywait not found — install inotify-tools" >&2
  exit 1
fi

echo "reload-helper: watching ${WATCH_DIR}, will run: ${RELOAD_CMD}"

while inotifywait -e close_write -r "${WATCH_DIR}" 2>/dev/null; do
  echo "reload-helper: cert change detected — running: ${RELOAD_CMD}"
  eval "${RELOAD_CMD}" || echo "reload-helper: reload command exited non-zero" >&2
done
