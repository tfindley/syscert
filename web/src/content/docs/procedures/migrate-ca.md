---
title: "SC-OPS-006: Migrate to a different CA"
navLabel: "006 · Migrate to a different CA"
description: Formal procedure for switching syscert from one certificate authority to another — editing the ACME config, validating offline, then forcing a fresh order against the new CA.
order: 6
eyebrow: "// docs · procedures · SC-OPS-006"
lede: Update the ACME block to point at the new CA, validate, then force a fresh order. A new CA means a new ACME account registration — the old certificate is not auto-revoked.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-006 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (config edit) and the `syscert` service user (renew/distribute steps) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Switch syscert from one certificate authority to another — for example, from Let's Encrypt to
an internal Vault PKI CA, from one internal CA to another, or from a custom ACME server to
Let's Encrypt. After the migration, the new CA issues all future certificates.

## Scope

Covers editing the `[acme]` block, handling EAB if the new CA requires it, and bootstrapping
connection trust for an untrusted internal CA. Does **not** automatically revoke the certificate
issued by the old CA — do that explicitly if required (see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

## Prerequisites

- syscert is installed and operational.
- You have the new CA's ACME directory URL (for `custom` CAs), contact email, and any EAB
  credentials. See [Configuration → `[acme]`](/docs/configuration/#acme--ca-and-challenge) for
  the `directory_url` values per CA type.
- Root access to edit `/etc/syscert/syscert.toml`.
- For an **untrusted internal CA**: the CA's root PEM available on disk to bootstrap connection
  trust (see step 2).

## Procedure

**1. Edit `[acme]` to point at the new CA.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Replace the existing `[acme]` block. Example — switching to a Vault PKI CA:

```toml
[acme]
ca            = "custom"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "you@example.com"
challenge     = "dns-01"
```

If the new CA requires **External Account Binding**, add `[acme.eab]` with the Key ID, and
put the HMAC key in `/etc/syscert/secrets` as `SYSCERT_EAB_HMAC=<base64url>`. See
[Configuration → `[acme.eab]`](/docs/configuration/#acmeeab--external-account-binding).

If the new CA is an **untrusted internal CA** (the host doesn't yet have its root in the
system trust store), set `acme.ca_bundle` to the path of its root PEM:

```toml
[acme]
ca_bundle = "/etc/syscert/my-internal-ca.pem"
```

This trusts the CA **for the ACME connection only** — it does not install the root
system-wide. To install it system-wide, run SC-OPS-008 afterwards.

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

This catches structural errors, unsupported combinations, and CA capability mismatches before
any network call is made. Fix all reported errors before continuing.

**3. Force a new ACME order against the new CA.**

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
```

syscert registers a **new ACME account** with the new CA (the old account is not reused), then
places an order and writes the new certificate to the store. If EAB was required, it is used
at account registration and not again.

**4. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

## Verification

Confirm the certificate was issued by the new CA:

```sh
openssl x509 -in /var/lib/syscert/cert.pem -noout -issuer
```

The `issuer` should identify the new CA. Also check the distributed copy:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -issuer -dates   # adjust to your path
```

If you set `acme.ca_bundle` for connection-only trust and want clients to trust the new
certificate without `-k`, install the CA root system-wide — see
[SC-OPS-008 — Trust an internal CA system-wide](/docs/procedures/trust-internal-ca/).

## Rollback / recovery

Revert the `[acme]` block to the previous CA configuration and run `renew --force` again:

```sh
sudo vi /etc/syscert/syscert.toml    # restore previous [acme] block
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
sudo -u syscert syscert distribute
```

The old CA will issue a fresh certificate. Note: if the old CA's ACME account key is still in
the store (`/var/lib/syscert/accounts/`), it will be reused. The certificate from the new CA
is not revoked automatically.

## Related procedures

- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — revoke the old CA's certificate explicitly.
- [SC-OPS-008 — Trust an internal CA system-wide](/docs/procedures/trust-internal-ca/) — install the new CA's root into the system trust store.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/) — wipe all state and re-provision from scratch.

**Explanatory docs:** [Configuration → `[acme]`](/docs/configuration/#acme--ca-and-challenge) ·
[Troubleshooting → x509 unknown authority](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca)
