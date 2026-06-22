---
title: "SC-OPS-004: Rotate the private key"
navLabel: "004 · Rotate the private key"
description: Formal procedure for rotating the certificate's private key — either via the default fresh-keypair-on-renewal behaviour or by disabling reuse_key before forcing a renewal.
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

Ensure the certificate is reissued with a new private key. By default, syscert already does
this on every renewal — a **fresh keypair is generated each time**. This procedure makes that
explicit and covers the `reuse_key = true` case where the key has been pinned.

## Scope

Covers two situations:

- **Default (`reuse_key = false`)**: a forced renewal already rotates the key — follow the
  short path below.
- **Key pinned (`reuse_key = true`)**: must clear the flag before renewing.

Does **not** revoke the certificate that held the old key — do that explicitly if required
(see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

## Prerequisites

- syscert is installed and the current certificate is valid.
- You know whether `reuse_key` is set in your `[cert]` block.
- Root access to edit `/etc/syscert/syscert.toml` (only if `reuse_key = true`).

## Procedure

### When `reuse_key` is `false` (the default)

A new keypair is generated automatically on every renewal. Force a renewal now:

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

A new keypair is generated. The old key is no longer used.

**4. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

**5. (Optional) Re-enable key pinning if required.**

If you needed a one-time rotation and want to re-pin the new key:

```sh
sudo vi /etc/syscert/syscert.toml    # set reuse_key = true again
```

## Verification

Confirm the private key fingerprint changed:

```sh
openssl pkey -in /var/lib/syscert/privkey.pem -noout -text 2>/dev/null | grep "Public-Key"
```

Compare the fingerprint against what was there before. A different key length or public key
value confirms rotation. To capture the new fingerprint for auditing:

```sh
openssl pkey -in /var/lib/syscert/privkey.pem -pubout | openssl dgst -sha256
```

Confirm the distributed copy was updated:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -enddate   # adjust to your path
```

## Rollback / recovery

The old key is not retained in the store after rotation (unless `[store] archive_keep > 0`,
in which case it is in `archive/<UTC-timestamp>/privkey.pem`). There is no automated
rollback path for a key rotation. If the new cert/key is problematic, force another renewal.

Note: rotating the key does **not** revoke the certificate that was bound to the old key. If
revocation is required, run [SC-OPS-005](/docs/procedures/revoke-and-replace/) instead.

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/) — renewal without key-rotation context.
- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoking the certificate that held the old key.
- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/) — changing key type (e.g. RSA → ECDSA).

**Explanatory docs:** [Configuration → `[cert]`](/docs/configuration/#cert--certificate-subject) ·
[Distributing certs](/docs/distributing/)
