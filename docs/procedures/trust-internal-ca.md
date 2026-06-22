---
title: "SC-OPS-008: Trust an internal CA system-wide"
navLabel: "008 · Trust an internal CA"
description: Formal procedure for installing or removing an internal CA's root certificate in the system trust store using syscert trust install and trust remove.
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

Install an internal CA's root certificate into the system-wide trust store so that standard
clients (curl, wget, browsers, OpenSSL) accept certificates issued by that CA without
disabling verification. Remove it when the CA is decommissioned or replaced.

For public CAs (Let's Encrypt), the system already trusts them — `trust install` exits
immediately with "nothing to do."

## Scope

Covers `syscert trust install` (add the root) and `syscert trust remove` (remove
SysCert-managed anchors). Both operations affect the system trust store and require `root`.

This is **separate from `acme.ca_bundle`**, which only trusts the CA for the ACME connection
to bootstrap issuance — it does not make clients trust the issued certificate. Use this
procedure when you want issued certificates to be trusted end-to-end without extra flags.

## Prerequisites

- Root access on the host.
- The internal CA's root PEM available on disk, **or** `acme.ca_bundle` already pointing to
  it in the config.

## Procedure

### Install the internal CA root

The CA source is resolved in order: `--ca-file <pem>` (if passed) → `acme.ca_bundle` in the
config. One of the two must be set.

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

The root is written to the system anchor directory and the trust database is updated (`update-ca-certificates` on Debian/Ubuntu; `update-ca-trust` on RHEL). Other syscert operations (`renew`, `distribute`) are unchanged.

---

### Remove SysCert-managed CA anchors

```sh
sudo syscert trust remove
```

This removes only anchors that were installed by `syscert trust install` — it does not touch
anchors managed by your OS or other tools.

On success:

```
OK: removed 1 SysCert-managed CA anchor(s) from /usr/local/share/ca-certificates
```

## Verification

After installing, confirm a client trusts a certificate issued by that CA without disabling
verification:

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

After removing, the same verification should fail with an untrusted-anchor error (confirming
the root is gone).

## Rollback / recovery

To re-install after a removal, re-run `trust install`. To undo an install, run `trust remove`.
Both operations are idempotent — re-running them is safe.

## Related procedures

- [SC-OPS-006 — Migrate to a different CA](/docs/procedures/migrate-ca/) — migrate the ACME config to a new CA first, then trust it here.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/) — destroy offers to remove managed CA anchors during a full teardown.
- [SC-OPS-001 — Install & deploy](/docs/procedures/install-and-deploy/) — initial setup, where this may be needed for an internal CA.

**Explanatory docs:** [Configuration → `acme.ca_bundle`](/docs/configuration/#acme--ca-and-challenge) ·
[Troubleshooting → x509 unknown authority](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca)
