---
title: Install manually
navLabel: Manually
description: Download a verified syscert release binary and set up the systemd service and timer by hand — the steps the one-line installer automates, one at a time.
order: 1
eyebrow: "// docs · advanced install · manually"
lede: Download a release binary, verify it, and run the systemd install yourself — the steps the one-liner automates, one at a time.
---

## Download a release binary & verify

Every release ships pre-built static binaries. Check them against the published
`sha256sums.txt` before you install:

```sh
# amd64 — for arm64 use syscert-linux-arm64
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/syscert-linux-amd64 -o syscert
chmod +x syscert

# verify against the published checksums
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/sha256sums.txt -o sha256sums.txt
sha256sum --check --ignore-missing sha256sums.txt

./syscert --help
```

Want a specific version? Swap `latest/download` for `download/<tag>` (say
`download/v0.3.0`). See
[all releases](https://github.com/tfindley/syscert/releases).

Prefer to build it yourself? See [Compile from source](/docs/advanced-install/compile-from-source/).

## Install as a systemd service

The installer lives **outside the binary**. The `syscert` binary never touches your
system; the script does, and you can run it as many times as you like. Point it at
the binary you downloaded or built:

```sh
# need the packaging files? clone the repo into a named dir (no Go required)
git clone https://github.com/tfindley/syscert.git syscert-src

# point the installer at your downloaded or built binary (idempotent; needs root)
sudo syscert-src/packaging/install.sh ./syscert
```

That single command does a lot. It creates the `syscert` system user and
`/var/lib/syscert` (`0700`), and installs the binary to `/usr/local/bin/syscert`. It
writes a starter `/etc/syscert/syscert.toml` (`0640 root:syscert`, kept out of
world-readable reach since it carries the internal CA URL and ACME email), a `0640`
`/etc/syscert/secrets`, and an `/etc/default/syscert` for operator settings — existing
files are never overwritten. Then it installs the units, enables the timer, and relabels
for SELinux where that's active. The binary gets relabeled too: `/usr/local/bin/syscert`
becomes `bin_t` so systemd can execute it on an enforcing host. If you place the binary
by hand instead of using the installer, run `sudo restorecon /usr/local/bin/syscert` to
set the right label.

> **`/usr/local/bin` and your PATH.** The systemd unit calls the absolute
> `/usr/local/bin/syscert`, so the service never leans on `PATH`. When you run it by hand,
> though, `/usr/local/bin` sometimes goes missing from a `sudo` `secure_path` or a
> minimal/`nologin` environment. If `syscert` isn't found, call `/usr/local/bin/syscert`
> directly or add the directory to your `PATH`.

No systemd on this host (an appliance or NAS)? Run it [as a cron job](/docs/advanced-install/cron/) instead.

## The user, service, and timer

syscert runs as a dedicated, no-login **system user**, never root. It owns the
canonical store and gets just one capability from the unit, `CAP_CHOWN`, so it can set
ownership on the copies it distributes. Add `CAP_NET_BIND_SERVICE` only if you use
http-01/tls-alpn-01 on :80/:443.

`syscert.service` is a hardened `oneshot` that runs bare `syscert` (issue/renew as
needed, then distribute):

```ini
[Unit]
Description=SysCert — ensure system TLS certificate is issued, renewed, and distributed
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
User=syscert
Group=syscert
EnvironmentFile=-/etc/default/syscert    # operator settings, e.g. SYSCERT_CONFIG; optional
EnvironmentFile=-/etc/syscert/secrets    # DNS/CA creds (0640); optional
ExecStart=/usr/local/bin/syscert         # bare syscert = issue/renew as needed, then distribute

# Hardening (abridged)
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/syscert
PrivateTmp=true
MemoryDenyWriteExecute=true

# CAP_CHOWN lets distribution set target ownership.
# Add CAP_NET_BIND_SERVICE only if you serve http-01/tls-alpn-01 on :80/:443.
AmbientCapabilities=CAP_CHOWN
CapabilityBoundingSet=CAP_CHOWN
```

There's **no long-running daemon**. The timer firing bare `syscert` *is* the
service: shortly after boot, then daily with jitter, and it catches up a missed run.
`install.sh` enables the timer but doesn't start it, so the first run can't fail against
the unconfigured starter config.

```ini
[Timer]
OnBootSec=5min
OnCalendar=daily
RandomizedDelaySec=12h
Persistent=true
```

> Once you've edited the config, start it: `sudo systemctl start syscert.timer`, then
> check with `systemctl list-timers syscert.timer`. An **Ansible role** for fleet
> installs is on the [roadmap](/docs/roadmap/), and it runs these same steps.

Removing it later? See [Uninstall](/docs/advanced-install/#uninstall).

---

Next: [Configuration](/docs/configuration/) · [Compile from source](/docs/advanced-install/compile-from-source/) ·
[As a cron job](/docs/advanced-install/cron/)
