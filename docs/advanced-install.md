---
title: Advanced install
navLabel: Advanced install
description: Install syscert by downloading and verifying a release binary, building from source, or doing the systemd steps by hand — plus the service/timer reference and uninstall.
order: 4
eyebrow: "// docs · advanced install"
lede: The one-liner is the fast path. Here are the verify-every-byte routes — download + checksum, build from source, manual systemd, and uninstall.
---

**Supported targets:** Debian/Ubuntu and the RHEL family (others may work but
aren't tested), on amd64 and arm64. For the one-line installer and an
inspect-first walkthrough, see the [install page](/install/); the steps below are
the building blocks it automates.

## Download a release binary & verify

Pre-built static binaries are published on every release. Verify them against the
published `sha256sums.txt` before installing:

```sh
# amd64 — for arm64 use syscert-linux-arm64
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/syscert-linux-amd64 -o syscert
chmod +x syscert

# verify against the published checksums
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/sha256sums.txt -o sha256sums.txt
sha256sum --check --ignore-missing sha256sums.txt

./syscert --help
```

Pin a specific version by swapping `latest/download` for `download/<tag>` (e.g.
`download/v0.0.6`). See
[all releases](https://github.com/tfindley/syscert/releases).

## Build from source

Requires **Go ≥ 1.26**. A local build derives its version from the checkout's VCS
info automatically (the tag, with a `+dirty` suffix when the tree has uncommitted
changes):

```sh
git clone https://github.com/tfindley/syscert.git
cd syscert
go build -o syscert ./cmd/syscert
./syscert version
```

## Install as a systemd service

The installer is **external to the binary** — the `syscert` binary never modifies
your system; the script does, and it's idempotent. Point it at your downloaded or
built binary:

```sh
# need the packaging files? clone the repo (no Go required)
git clone https://github.com/tfindley/syscert.git

# point the installer at your downloaded or built binary (idempotent; needs root)
sudo packaging/install.sh ./syscert
```

It creates the `syscert` system user and `/var/lib/syscert` (`0700`), installs the
binary to `/usr/local/bin/syscert`, writes a starter `/etc/syscert/syscert.toml`,
a `0640` `/etc/syscert/secrets`, and an `/etc/default/syscert` for operator
settings (never overwriting existing files), installs the units, enables the
timer, and relabels for SELinux where active.

## Uninstall

Installed with the one-liner? Remove it the same way — no clone needed:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall          # keep data
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge  # + data, config, user
```

From a source checkout, use the script directly:

```sh
sudo packaging/install.sh --uninstall            # remove units + binary, keep data
sudo packaging/install.sh --uninstall --purge    # also remove /var/lib/syscert, /etc/syscert, user
```

`--purge` is irreversible, so it asks you to confirm on the terminal; set
`SYSCERT_ASSUME_YES=1` to skip the prompt (e.g. in automation).

## The user, service, and timer

syscert runs as a dedicated, no-login **system user** — never root. It owns the
canonical store and is granted only `CAP_CHOWN` (via the unit) so it can set
ownership on the copies it distributes; add `CAP_NET_BIND_SERVICE` only if you use
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

There is **no long-running daemon** — the timer firing bare `syscert` *is* the
service: shortly after boot, then daily with jitter, catching up a missed run.
`install.sh` enables but does not start it, so the first run can't fail against the
unconfigured starter config.

```ini
[Timer]
OnBootSec=5min
OnCalendar=daily
RandomizedDelaySec=12h
Persistent=true
```

> Start it after editing the config: `sudo systemctl start syscert.timer`, then
> check with `systemctl list-timers syscert.timer`. An **Ansible role** for fleet
> installs is on the [roadmap](/docs/roadmap/); it performs these same steps.

---

Next: [Quick start](/docs/quick-start/) · [Configuration](/docs/configuration/) ·
[Sample configurations](/docs/examples/)
