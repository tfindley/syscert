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

Switch syscert from one certificate authority to another. That might be Let's Encrypt to an
internal Vault PKI CA, one internal CA to another, or a custom ACME server back to Let's Encrypt.
From then on, the new CA issues every certificate.

## Scope

Covers editing the `[acme]` block, handling EAB when the new CA needs it, and bootstrapping
connection trust for an untrusted internal CA. It won't revoke the old CA's certificate for you;
do that explicitly if you need to (see [SC-OPS-005](/docs/procedures/revoke-and-replace/)).

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

Replace the existing `[acme]` block. Here's a switch to a Vault PKI CA:

```toml
[acme]
ca            = "custom"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "you@example.com"
challenge     = "dns-01"
```

If the new CA requires **External Account Binding**, add `[acme.eab]` with the Key ID and put
the HMAC key in `/etc/syscert/secrets` as `SYSCERT_EAB_HMAC=<base64url>`. See
[Configuration → `[acme.eab]`](/docs/configuration/#acmeeab--external-account-binding).

If the new CA is an **untrusted internal CA** (the host doesn't have its root in the system
trust store yet), point `acme.ca_bundle` at the root PEM on disk:

```toml
[acme]
ca_bundle = "/etc/syscert/my-internal-ca.pem"
```

That trusts the CA **for the ACME connection only**. It doesn't install the root system-wide.
To do that, run SC-OPS-008 afterwards.

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

This runs before any network call. It catches structural errors, unsupported combinations, and
cases where the CA can't do what the config asks. Fix every error it reports before you continue.

**3. Force a new ACME order against the new CA.**

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
```

syscert registers a **new ACME account** with the new CA (it doesn't reuse the old account),
then places an order and writes the new certificate to the store. If EAB was required, it's
used at account registration and never again.

**4. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

## Verification

Confirm the certificate was issued by the new CA:

```sh
openssl x509 -in /var/lib/syscert/cert.pem -noout -issuer
```

The `issuer` should name the new CA. Check the distributed copy too:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -issuer -dates   # adjust to your path
```

If you set `acme.ca_bundle` for connection-only trust and you want clients to trust the new
certificate without `-k`, install the CA root system-wide. See
[SC-OPS-008 — Trust an internal CA system-wide](/docs/procedures/trust-internal-ca/).

## Rollback / recovery

Put the `[acme]` block back to the previous CA and run `renew --force` again:

```sh
sudo vi /etc/syscert/syscert.toml    # restore previous [acme] block
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
sudo -u syscert syscert distribute
```

The old CA issues a fresh certificate. One catch: if its ACME account key is still in the store
(`/var/lib/syscert/accounts/`), syscert reuses it. The certificate from the new CA isn't revoked
automatically.

## Related procedures

- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/): revoke the old CA's certificate explicitly.
- [SC-OPS-008 — Trust an internal CA system-wide](/docs/procedures/trust-internal-ca/): install the new CA's root into the system trust store.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/): wipe all state and re-provision from scratch.

**Explanatory docs:** [Configuration → `[acme]`](/docs/configuration/#acme--ca-and-challenge) ·
[Troubleshooting → x509 unknown authority](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca)
