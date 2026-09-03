#!/bin/sh
# syscert + nginx entrypoint — restart-on-failure mitigation for the embedded pattern.
# ------------------------------------------------------------------------------
# Starts syscert (--interval 12h) and nginx. If EITHER process exits (for any
# reason), this script exits too — so Docker's restart policy re-runs the whole
# container. This surfaces failures rather than silently serving a stale cert.
#
# For a more robust two-process setup, replace this script with a real process
# supervisor (s6-overlay, runit, or tini + a process group). This script is
# intentionally minimal.
#
# Secrets: load DNS-provider credentials from the environment. Pass them at
# runtime with `--env-file` or Docker secrets — never bake them in the image.
# ------------------------------------------------------------------------------

set -e

# Validate the config before starting anything (exits non-zero on error).
syscert dry-run --config-only --config /etc/syscert/syscert.toml

# Start nginx in the foreground (background it ourselves so we can monitor both).
nginx -g "daemon off;" &
NGINX_PID=$!

# Start syscert's renewal loop. It runs as the syscert user (set in Dockerfile).
# dns-01 is the challenge — no ports needed.
syscert --interval 12h --config /etc/syscert/syscert.toml &
SYSCERT_PID=$!

# Reload nginx each time syscert writes a new cert (Linux only — inotifywait).
# Remove this block if inotifywait is not in the image; use a periodic cron instead.
if command -v inotifywait >/dev/null 2>&1; then
  (
    while inotifywait -e close_write /var/lib/syscert 2>/dev/null; do
      nginx -s reload
    done
  ) &
fi

# Wait for EITHER process to exit, then exit so Docker restarts the container.
#
# Not `wait -n`: this runs under /bin/sh, which on nginx:alpine is BusyBox ash,
# where -n is undefined (shellcheck SC3045). The tempting
# `wait -n … || wait …` fallback is worse than useless — it silently waits for
# BOTH processes, so a dead nginx alongside a live syscert would never restart
# the container, which is the exact failure this entrypoint exists to catch.
# Polling with kill -0 is plain POSIX and behaves identically everywhere.
while kill -0 "$NGINX_PID" 2>/dev/null && kill -0 "$SYSCERT_PID" 2>/dev/null; do
  sleep 5
done
