---
title: "SC-OPS-002: Change certificate details & reissue"
navLabel: "002 · Change cert details & reissue"
description: How to change the certificate's identity, key type, or issuance profile and force a fresh certificate from the CA.
order: 2
eyebrow: "// docs · procedures · SC-OPS-002"
lede: Edit the config, validate offline, then force a new ACME order and distribute. The timer won't spot config changes on its own, so this is how you apply them.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-002 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (config edit) and the `syscert` service user (renew/distribute steps) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Apply a change to the certificate's identity or issuance parameters, then force a fresh certificate
from the CA that reflects it. In scope: adding or removing Subject Alternative Names, changing the
key type, switching the ACME profile.

## Scope

Covers changes to any of:

- `[cert] hostname`, `[cert] sans`, `[cert] ip_sans`
- `[cert] key_type`, `[cert] reuse_key`
- `[acme] profile`, `[acme] challenge`

It doesn't cover switching to a different CA (see SC-OPS-006 when available), or revoking the
current certificate before you reissue it (see SC-OPS-005 when available).

## Prerequisites

- syscert is already installed and the current certificate is valid.
- You know the new values you want to set. Check [Configuration](/docs/configuration/) for the
  allowed keys and their constraints (e.g. `ip_sans` forces the challenge to `http-01`/`tls-alpn-01`;
  private IPs require an internal CA).
- Root access to edit `/etc/syscert/syscert.toml`.

## Procedure

**1. Edit the configuration.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Make your change. Here's one that adds an IP SAN:

```toml
[cert]
hostname = "host.example.com"
sans     = ["api.example.com"]
ip_sans  = ["10.0.1.5"]          # forces challenge to http-01 or tls-alpn-01
```

The full key reference and its constraints live in
[Configuration → `[cert]`](/docs/configuration/#cert--certificate-subject).

**2. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

This runs before any network call. It catches structural problems, unsupported combinations, and
cases where the CA can't do what the config asks. Fix every error it reports before you continue.

**3. Force a new ACME order.**

```sh
sudo -u syscert syscert renew --force
```

`--force` skips the expiry check and places a new order from the current config, then writes the
fresh certificate to the store (`/var/lib/syscert/`). You have to do this by hand: the timer's
automatic `ensure`/`renew` is **expiry-driven only** and never looks at config changes.

By default you get a fresh keypair (`reuse_key = false`). The old certificate is overwritten in the
store. If you've set `[store] archive_keep`, the previous set is snapshotted first.

**4. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

`renew --force` writes the store, but it won't deliver to the paths in your `[[distribute]]` blocks.
That's what this separate `distribute` step is for. Afterwards every target path holds the new
certificate.

## Verification

Check that the new certificate carries the SANs you expect:

```sh
openssl x509 -in /var/lib/syscert/cert.pem -noout -text | grep -A1 "Subject Alternative Name"
```

Expected output (adjust for your own names and IPs):

```
X509v3 Subject Alternative Name:
    DNS:host.example.com, DNS:api.example.com, IP Address:10.0.1.5
```

Then confirm the distributed copies changed too:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -enddate   # adjust to your path
```

The `notAfter` date should match the certificate you just issued.

## Rollback / recovery

Every run here issues a fresh keypair and certificate, so reverting just means going back to the old config:

1. Restore the previous `syscert.toml` (from a backup or version control).
2. Run `renew --force` again to reissue from that config:

```sh
sudo vi /etc/syscert/syscert.toml    # restore previous config
sudo -u syscert syscert renew --force
sudo -u syscert syscert distribute
```

The CA issues a new certificate matching the restored config. It won't revoke the old one for you;
do that explicitly if you need it (see SC-OPS-005 when available).

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/): a new cert with no
  config change (expiry bypass only).
- [SC-OPS-004 — Rotate the private key](/docs/procedures/rotate-key/): rotate the key only, leaving
  the certificate identity alone.
- [SC-OPS-006 — Migrate to a different CA](/docs/procedures/migrate-ca/): switch CA entirely.

**Explanatory docs:** [Configuration](/docs/configuration/) · [Distributing certs](/docs/distributing/) ·
[Troubleshooting](/docs/troubleshooting/)
