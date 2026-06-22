---
title: "SC-OPS-010: Uninstall or purge"
navLabel: "010 · Uninstall or purge"
description: Formal procedure for removing syscert from a host — keeping data with --uninstall or fully purging config, store, and the syscert user with --uninstall --purge.
order: 10
eyebrow: "// docs · procedures · SC-OPS-010"
lede: --uninstall removes the binary and units but keeps config and certificates. Add --purge to also remove the store, config, and the syscert user.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-010 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` |
| **Last reviewed** | 2026-06-22 |

## Purpose

Remove syscert from a host cleanly — either preserving certificate state for later
reinstallation (`--uninstall`) or removing everything including the store, config files, and
the `syscert` system user (`--uninstall --purge`).

## Scope

Covers the two uninstall modes driven by `packaging/install.sh`. Does **not** revoke the
certificate before removal — do that first if needed (see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

## Prerequisites

- Root access on the host.
- If you want to revoke the certificate before removing, run [SC-OPS-005](/docs/procedures/revoke-and-replace/) first.

## Procedure

### Via the one-line installer

**Keep config and certificates (binary + units removed):**

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall
```

**Remove everything, including `/var/lib/syscert`, `/etc/syscert`, and the `syscert` user:**

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge
```

### Via a local copy of the installer

```sh
sudo packaging/install.sh --uninstall
sudo packaging/install.sh --uninstall --purge
```

### Confirmation and automation

`--purge` asks for confirmation before removing the store and user:

```
Remove /var/lib/syscert, /etc/syscert, and the syscert user? [y/N]
```

To skip the prompt in automation:

```sh
SYSCERT_ASSUME_YES=1 curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge
```

## What each mode removes

| Item | `--uninstall` | `--uninstall --purge` |
|---|---|---|
| `/usr/local/bin/syscert` | removed | removed |
| `/etc/systemd/system/syscert.service` | removed | removed |
| `/etc/systemd/system/syscert.timer` | removed | removed |
| `/etc/default/syscert` | removed | removed |
| `/etc/syscert/syscert.toml` | **kept** | removed |
| `/etc/syscert/secrets` | **kept** | removed |
| `/var/lib/syscert/` (store, certs, account) | **kept** | removed |
| `syscert` system user | **kept** | removed |

## Verification

After `--uninstall`:

```sh
which syscert                        # should return nothing
systemctl list-timers syscert.timer  # should show no timer
ls /var/lib/syscert/                 # store is intact
```

After `--uninstall --purge`:

```sh
ls /var/lib/syscert/     # no such directory
ls /etc/syscert/         # no such directory
id syscert               # no such user
```

## Rollback / recovery

- After `--uninstall` (data kept): re-run the installer to restore the binary and units. The
  existing config and certificates are reused.
- After `--uninstall --purge`: there is no recovery path — the store is gone. Reinstall from
  scratch using [SC-OPS-001](/docs/procedures/install-and-deploy/).

## Related procedures

- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoke the certificate before uninstalling if required.
- [SC-OPS-001 — Install & deploy](/docs/procedures/install-and-deploy/) — reinstall after an `--uninstall` (data-preserving) removal.
- [SC-OPS-009 — Upgrade syscert](/docs/procedures/upgrade/) — in-place binary swap without removing the installation.

**Explanatory docs:** [Advanced install](/docs/advanced-install/) · [Quick start](/docs/quick-start/)
