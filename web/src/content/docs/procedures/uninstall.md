---
title: "SC-OPS-010: Uninstall or purge"
navLabel: "010 · Uninstall or purge"
description: How to remove syscert from a host, keeping data with --uninstall or fully purging config, store, and the syscert user with --uninstall --purge.
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

Take syscert off a host cleanly. Either keep the certificate state for a later reinstall (`--uninstall`), or wipe everything: the store, the config files, and the `syscert` system user (`--uninstall --purge`).

## Scope

Covers the two uninstall modes that `packaging/install.sh` drives. It does **not** revoke the certificate before removal, so do that first if you need it (see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

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

`--purge` warns and asks you to confirm before it removes anything. It wants the whole word — `y` aborts:

```
WARNING: --purge will PERMANENTLY delete:
    /var/lib/syscert   (private keys + certificates)
    /etc/syscert        (config + secrets)
    the syscert system user and group
Type 'yes' to continue:
```

To skip the prompt in automation, the variable has to reach the shell running the script, not `curl` — `sudo` also resets the environment, so set it after `sudo`:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo SYSCERT_ASSUME_YES=1 sh -s -- --uninstall --purge
```

With no terminal to confirm on and no `SYSCERT_ASSUME_YES`, `--purge` refuses rather than guessing.

## What each mode removes

| Item | `--uninstall` | `--uninstall --purge` |
|---|---|---|
| `/usr/local/bin/syscert` | removed | removed |
| `/etc/systemd/system/syscert.service` | removed | removed |
| `/etc/systemd/system/syscert.timer` | removed | removed |
| `/etc/default/syscert` | **kept** | removed |
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

- After `--uninstall` (data kept): run the installer again to bring back the binary and units. Your existing config and certificates get reused.
- After `--uninstall --purge`: there's no way back. The store is gone, so reinstall from scratch with [SC-OPS-001](/docs/procedures/install-and-deploy/).

## Related procedures

- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoke the certificate before uninstalling if required.
- [SC-OPS-001 — Install & deploy](/docs/procedures/install-and-deploy/) — reinstall after an `--uninstall` (data-preserving) removal.
- [SC-OPS-009 — Upgrade syscert](/docs/procedures/upgrade/) — in-place binary swap without removing the installation.

**Explanatory docs:** [Advanced install](/docs/advanced-install/) · [Quick start](/docs/quick-start/)
