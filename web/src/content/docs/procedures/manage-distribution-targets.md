---
title: "SC-OPS-007: Manage distribution targets"
navLabel: "007 · Manage distribution targets"
description: How to add, change, or remove [[distribute]] blocks and push the updated delivery config to disk.
order: 7
eyebrow: "// docs · procedures · SC-OPS-007"
lede: Edit the [[distribute]] blocks, validate offline, then run distribute. No reissue needed; distribute reads from the store you already have.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-007 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (config edit) and the `syscert` service user (distribute step) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Add a new delivery target, change an existing target's path, ownership, mode, or SELinux context, or stop delivering to a path. All without reissuing the certificate.

## Scope

Covers changes to `[[distribute]]` blocks and nothing else. The certificate in the store doesn't change. What changes is what gets copied, where it lands, and with which permissions.

It doesn't cover reissuing the certificate. If you also need a new cert, run [SC-OPS-003](/docs/procedures/force-renewal/) (or [SC-OPS-002](/docs/procedures/change-cert-details/) for identity changes) after this one.

## Prerequisites

- syscert is installed and a valid certificate is in the store.
- Root access to edit `/etc/syscert/syscert.toml`.
- For a new target path: the destination directory exists and the `syscert` user has write access (or `CAP_CHOWN` grants it; the shipped unit grants that capability).

## Procedure

**1. Edit the `[[distribute]]` blocks.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Add, change, or remove `[[distribute]]` blocks. Each block copies one artifact to one path. Here's a pair: an nginx target and an application bundle target:

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

Key constraints, enforced at validation and at runtime:

- `privkey` and `bundle` carry the private key, so a world-readable mode like `0644` is **rejected up front**.
- `selinux_context` only applies on RHEL-family hosts with SELinux active; on Debian/Ubuntu it's a no-op.
- The full key reference is in [Configuration → `[[distribute]]`](/docs/configuration/#distribute--delivering-to-consumers).

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

This catches mode and artifact constraint violations before anything gets written. Fix every error it reports before you continue.

**3. Run distribute.**

```sh
sudo -u syscert syscert distribute
```

`distribute` reads the current certificate from the store and copies it to every configured target. No ACME call, no reissue.

## A note on removing targets

Removing a `[[distribute]]` block from the config stops future writes to that path. It does **not** delete the file that was already written. Remove that yourself if you need to:

```sh
sudo rm /path/to/old/target.pem
```

## Verification

Check that each new or changed target has the file, owner, mode, and (on RHEL) SELinux context you expect:

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

Files written to new paths during the change aren't cleaned up for you. Remove them by hand if you're fully reverting.

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/): if you also need a fresh certificate.
- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/): if you need to change what the certificate contains too.

**Explanatory docs:** [Distributing certs](/docs/distributing/) · [Configuration → `[[distribute]]`](/docs/configuration/#distribute--delivering-to-consumers)
