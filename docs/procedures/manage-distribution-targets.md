---
title: "SC-OPS-007: Manage distribution targets"
navLabel: "007 · Manage distribution targets"
description: Formal procedure for adding, changing, or removing [[distribute]] blocks and pushing the updated delivery configuration to disk.
order: 7
eyebrow: "// docs · procedures · SC-OPS-007"
lede: Edit the [[distribute]] blocks, validate offline, then run distribute. No reissue required — distribute reads from the existing store.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-007 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (config edit) and the `syscert` service user (distribute step) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Add a new delivery target, change the path, ownership, mode, or SELinux context of an existing
target, or stop delivering to a path — without reissuing the certificate.

## Scope

Covers changes to `[[distribute]]` blocks only. The certificate in the store is unchanged;
only what gets copied, where, and with what permissions is affected.

Does **not** cover reissuing the certificate — if you also need a new cert, run
[SC-OPS-003](/docs/procedures/force-renewal/) (or [SC-OPS-002](/docs/procedures/change-cert-details/)
for identity changes) after this procedure.

## Prerequisites

- syscert is installed and a valid certificate is in the store.
- Root access to edit `/etc/syscert/syscert.toml`.
- For a new target path: the destination directory exists and the `syscert` user has write
  access (or `CAP_CHOWN` grants it — the shipped unit grants this capability).

## Procedure

**1. Edit the `[[distribute]]` blocks.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Add, change, or remove `[[distribute]]` blocks. Each block copies one artifact to one path.
Example — adding an nginx target and an application bundle target:

```toml
[[distribute]]
artifact = "fullchain"
path     = "/etc/nginx/tls/fullchain.pem"
owner    = "root"
group    = "root"
mode     = "0644"

[[distribute]]
artifact = "privkey"
path     = "/etc/nginx/tls/privkey.pem"
owner    = "root"
group    = "root"
mode     = "0600"

[[distribute]]
artifact        = "bundle"
path            = "/etc/someapp/tls/combined.pem"
owner           = "someapp"
group           = "someapp"
mode            = "0600"
selinux_context = "cert_t"
```

Key constraints (enforced at validation and runtime):

- `privkey` and `bundle` contain the private key — a world-readable mode (e.g. `0644`) is
  **rejected up front**.
- `selinux_context` is only applied on RHEL-family hosts with SELinux active; it is a no-op
  on Debian/Ubuntu.
- See [Configuration → `[[distribute]]`](/docs/configuration/#distribute--delivering-to-consumers)
  for the full key reference.

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

This catches mode/artifact constraint violations before writing anything. Fix all reported
errors before continuing.

**3. Run distribute.**

```sh
sudo -u syscert syscert distribute
```

`distribute` reads the current certificate from the store and copies it to every configured
target. No ACME call is made; the certificate is not reissued.

## A note on removing targets

Removing a `[[distribute]]` block from the config stops future writes to that path. It does
**not** delete the file that was already written — you must remove it manually if needed:

```sh
sudo rm /path/to/old/target.pem
```

## Verification

Confirm each new or changed target has the expected file, owner, mode, and (on RHEL) SELinux
context:

```sh
ls -l /etc/nginx/tls/fullchain.pem
ls -l /etc/nginx/tls/privkey.pem
```

On RHEL-family hosts, verify the SELinux context:

```sh
ls -Z /etc/someapp/tls/combined.pem
```

## Rollback / recovery

Restore the previous `[[distribute]]` config and re-run `distribute`:

```sh
sudo vi /etc/syscert/syscert.toml    # restore previous blocks
sudo -u syscert syscert distribute
```

Files written to new paths during the attempted change are not automatically cleaned up —
remove them manually if the change is being fully reverted.

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/) — if you also need a fresh certificate.
- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/) — if you need to change what the certificate contains as well.

**Explanatory docs:** [Distributing certs](/docs/distributing/) ·
[Configuration → `[[distribute]]`](/docs/configuration/#distribute--delivering-to-consumers)
