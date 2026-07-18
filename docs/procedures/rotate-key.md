---
title: "SC-OPS-004: Rotate the private key"
navLabel: "004 · Rotate the private key"
description: How to rotate the certificate's private key, either through the default fresh-keypair-on-renewal behaviour or by clearing reuse_key before you force a renewal.
order: 4
eyebrow: "// docs · procedures · SC-OPS-004"
lede: syscert generates a fresh keypair on every renewal by default. If reuse_key is set, clear it first, then force a renewal.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-004 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (config edit if needed) and the `syscert` service user (renew/distribute steps) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Reissue the certificate with a new private key. syscert already does this on every renewal by
default: a **fresh keypair is generated each time**. This procedure spells that out and also
handles the `reuse_key = true` case, where you've pinned the key.

## Scope

Two situations. If `reuse_key = false` (the default), a forced renewal already rotates the
key, so take the short path below. If `reuse_key = true`, the key is pinned and you have to
clear that flag before renewing.

This does **not** revoke the certificate that held the old key. Do that separately if you need
it (see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

## Prerequisites

- syscert is installed and the current certificate is valid.
- You know whether `reuse_key` is set in your `[cert]` block.
- Root access to edit `/etc/syscert/syscert.toml` (only if `reuse_key = true`).

## Procedure

### When `reuse_key` is `false` (the default)

Every renewal generates a new keypair on its own. Force one now:

**1. Force a renewal.**

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
```

**2. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

Skip to **Verification** below.

---

### When `reuse_key = true`

**1. Edit the configuration to disable key reuse.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Change:

```toml
[cert]
reuse_key = true
```

to:

```toml
[cert]
reuse_key = false
```

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

**3. Force a renewal.**

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
```

You get a new keypair, and the old key is no longer used.

**4. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

**5. (Optional) Re-enable key pinning if required.**

If this was a one-off rotation and you want to pin the new key again:

```sh
sudo vi /etc/syscert/syscert.toml    # set reuse_key = true again
```

## Verification

Confirm the private key fingerprint changed:

```sh
openssl pkey -in /var/lib/syscert/privkey.pem -noout -text 2>/dev/null | grep "Public-Key"
```

Compare that against whatever you had before. A different key length or public-key value means
the rotation took. To capture the new fingerprint for your audit trail:

```sh
openssl pkey -in /var/lib/syscert/privkey.pem -pubout | openssl dgst -sha256
```

Confirm the distributed copy was updated:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -enddate   # adjust to your path
```

## Rollback / recovery

Once you rotate, the store drops the old key, unless `[store] archive_keep > 0`, in which case
you'll find it under `archive/<UTC-timestamp>/privkey.pem`. There's no automated rollback for a
key rotation. If the new cert or key gives you trouble, force another renewal.

Note: rotating the key does **not** revoke the certificate that was bound to the old one. If
you need revocation, run [SC-OPS-005](/docs/procedures/revoke-and-replace/) instead.

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/) — renewal without key-rotation context.
- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoking the certificate that held the old key.
- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/) — changing key type (e.g. RSA → ECDSA).

**Explanatory docs:** [Configuration → `[cert]`](/docs/configuration/#cert--certificate-subject) ·
[Distributing certs](/docs/distributing/)
