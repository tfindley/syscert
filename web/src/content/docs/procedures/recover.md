---
title: "SC-OPS-011: Recover from a broken state"
navLabel: "011 · Recover from a broken state"
description: Formal procedure for wiping all syscert certificate state and ACME account data and re-provisioning from scratch using the destroy subcommand.
order: 11
eyebrow: "// docs · procedures · SC-OPS-011"
lede: destroy wipes the cert artifacts and ACME account (or just the certs with --keep-account), does NOT revoke, and does NOT reissue. Then run bare syscert to re-provision.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-011 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `syscert` service user (destroy + re-provision) · `root` (if un-trusting the CA root) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Wipe all certificate material and ACME account state from the store so syscert can be
re-provisioned from a clean slate. Use this when the store is corrupted or in an inconsistent
state, when switching CA (and a clean break is preferred), or when you need to re-register an
ACME account.

## Scope

Covers the `destroy` subcommand, which wipes the store and — optionally — offers to remove
system-trust anchors for an internal CA. Does **not** revoke the current certificate: if the
certificate must be revoked first, run [SC-OPS-005](/docs/procedures/revoke-and-replace/)
**before** this procedure.

## Prerequisites

- syscert is installed and the `syscert` user can access the store.
- If the current certificate must be revoked: run [SC-OPS-005](/docs/procedures/revoke-and-replace/) first, then return here.
- If switching CA and keeping the ACME account (same CA, new account not needed): use `--keep-account`.
- If the new CA requires EAB: have the new Key ID and HMAC ready before re-provisioning.

## Procedure

**1. (Optional) Revoke the current certificate first.**

`destroy` does **not** revoke. If revocation is required:

```sh
sudo -u syscert syscert void --env-file /etc/syscert/secrets
```

Then return here.

**2. Destroy cert state.**

Interactive (recommended — reads the prompt carefully):

```sh
sudo -u syscert syscert destroy
```

You will be prompted:

```
Destroy SysCert state in /var/lib/syscert (certificate + ACME account)?
This does NOT revoke the current cert — run `void` first if you need that. [y/N]
```

Enter `y`. To skip the prompt:

```sh
sudo -u syscert syscert destroy --force
```

**With `--keep-account`** (removes cert artifacts only; keeps the ACME account key and
registration so re-provisioning does not require a new EAB token):

```sh
sudo -u syscert syscert destroy --keep-account
sudo -u syscert syscert destroy --keep-account --force
```

**Internal CA only — un-trust the CA root.** On a full destroy (without `--keep-account`),
if the configured CA is an internal CA (`ca = "custom"`), `destroy` asks:

```
Also remove the internal CA from the system trust store (requires root)? [y/N]
```

This step requires `root`. If you ran `destroy` as the `syscert` user and want to un-trust
the CA root separately, use `trust remove`:

```sh
sudo syscert trust remove
```

**3. (Optional) Update the config if needed.**

If you are switching CA or changing other settings, edit the config now:

```sh
sudo vi /etc/syscert/syscert.toml
```

Validate offline:

```sh
sudo -u syscert syscert dry-run --config-only
```

**4. Re-provision.**

Run bare `syscert` (the `ensure` default — issue if no cert exists, then distribute):

```sh
sudo -u syscert syscert --env-file /etc/syscert/secrets
```

syscert registers a new ACME account (unless `--keep-account` was used), places a new order,
writes the certificate to the store, and distributes to all configured targets.

## Verification

```sh
sudo -u syscert syscert status
```

Confirm the cert subject, expiry, account, and distribution targets are as expected. Also
check a distributed path:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -dates   # adjust to your path
```

## Rollback / recovery

`destroy` is irreversible — the store contents are removed. Recovery is re-provisioning (step
4). If the previous ACME account was destroyed and the new CA requires EAB, obtain a new EAB
token before re-provisioning and update `/etc/syscert/secrets` with `SYSCERT_EAB_HMAC=<new>`.

## Related procedures

- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoke before destroying if the certificate must be invalidated.
- [SC-OPS-006 — Migrate to a different CA](/docs/procedures/migrate-ca/) — migrate the ACME config (preserve the store; no destroy needed in most cases).
- [SC-OPS-008 — Trust an internal CA system-wide](/docs/procedures/trust-internal-ca/) — re-install the CA root after a re-provision to a new internal CA.
- [SC-OPS-010 — Uninstall or purge](/docs/procedures/uninstall/) — full removal of the binary and files, not just the store.

**Explanatory docs:** [Troubleshooting](/docs/troubleshooting/) · [Configuration](/docs/configuration/)
