---
title: "SC-OPS-008: Trust an internal CA system-wide"
navLabel: "008 · Trust an internal CA"
description: How to install or remove an internal CA's root certificate in the system trust store with syscert trust install and trust remove.
order: 8
eyebrow: "// docs · procedures · SC-OPS-008"
lede: trust install adds the internal CA's root to the system trust store so clients accept its certificates without -k. trust remove removes it. Both require root.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-008 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` |
| **Last reviewed** | 2026-06-22 |

## Purpose

Put an internal CA's root certificate into the system-wide trust store so standard clients (curl, wget, browsers, OpenSSL) accept that CA's certificates without you having to turn off verification. Pull it back out when the CA is retired or replaced.

For public CAs like Let's Encrypt, the system already trusts them, so `trust install` just exits with "nothing to do."

## Scope

Covers `syscert trust install` (add the root) and `syscert trust remove` (pull SysCert-managed anchors back out). Both touch the system trust store, and both need `root`.

This is **separate from `acme.ca_bundle`**. That setting only trusts the CA for the ACME connection so issuance can bootstrap; it doesn't make clients trust the certificate you get out the other end. Use this procedure when you want issued certificates trusted end-to-end without extra flags.

## Prerequisites

- Root access on the host.
- The internal CA's root PEM available on disk, **or** `acme.ca_bundle` already pointing to it in the config.

## Procedure

### Install the internal CA root

syscert works out where to read the CA from in order: `--ca-file <pem>` if you pass it, otherwise `acme.ca_bundle` from the config. One of the two has to be set.

**Option A — source from `acme.ca_bundle` (already in config):**

```sh
sudo syscert trust install
```

**Option B — source from an explicit PEM file:**

```sh
sudo syscert trust install --ca-file /path/to/internal-ca.pem
```

On a public CA (`ca = "letsencrypt"`), the command exits immediately:

```
this is a public CA — the system already trusts it; nothing to do.
```

On success:

```
OK: installed CA "My Internal CA" into the system trust store (/usr/local/share/ca-certificates)
```

syscert writes the root into the system anchor directory and refreshes the trust database (`update-ca-certificates` on Debian/Ubuntu, `update-ca-trust` on RHEL). Nothing else changes: `renew` and `distribute` carry on as before.

---

### Remove SysCert-managed CA anchors

```sh
sudo syscert trust remove
```

This only removes anchors that `syscert trust install` put there. It leaves anchors managed by your OS or other tools alone.

On success:

```
OK: removed 1 SysCert-managed CA anchor(s) from /usr/local/share/ca-certificates
```

## Verification

Once installed, check that a client trusts a certificate from that CA without you disabling verification:

```sh
curl https://host.example.com
```

Or verify the chain explicitly:

```sh
openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt /var/lib/syscert/cert.pem
```

Expected output:

```
/var/lib/syscert/cert.pem: OK
```

After a remove, that same check should fail with an untrusted-anchor error, which tells you the root is gone.

## Rollback / recovery

To reinstall after a removal, run `trust install` again. To undo an install, run `trust remove`. Both are idempotent, so running them twice does no harm.

## Related procedures

- [SC-OPS-006 — Migrate to a different CA](/docs/procedures/migrate-ca/) — migrate the ACME config to a new CA first, then trust it here.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/) — destroy offers to remove managed CA anchors during a full teardown.
- [SC-OPS-001 — Install & deploy](/docs/procedures/install-and-deploy/) — initial setup, where this may be needed for an internal CA.

**Explanatory docs:** [Configuration → `acme.ca_bundle`](/docs/configuration/#acme--ca-and-challenge) · [Troubleshooting → x509 unknown authority](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca)
