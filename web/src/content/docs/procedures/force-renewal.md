---
title: "SC-OPS-003: Force an immediate renewal"
navLabel: "003 · Force an immediate renewal"
description: Formal procedure for renewing the certificate immediately — bypassing the expiry window — without making any configuration change.
order: 3
eyebrow: "// docs · procedures · SC-OPS-003"
lede: Renew the certificate right now, regardless of how much time is left. No config change is needed — this is the expiry-bypass path only.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-003 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `syscert` service user (renew/distribute steps) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Place a new ACME order and replace the certificate immediately, without modifying the
configuration and without waiting for the timer's expiry-driven renewal window.

## Scope

Covers the case where the certificate is valid but you need a fresh one now — for example,
after a CA incident, a missed distribution, or a service restart that reloaded stale cached
material. The configuration does **not** change.

Does **not** cover:

- Changes to certificate identity (SANs, key type, etc.) — see [SC-OPS-002](/docs/procedures/change-cert-details/).
- Revoking the current certificate before reissuing — see [SC-OPS-005](/docs/procedures/revoke-and-replace/).
- Key-only rotation — see [SC-OPS-004](/docs/procedures/rotate-key/).

## Prerequisites

- syscert is installed and the current certificate is valid.
- The store at `/var/lib/syscert` is accessible to the `syscert` user.
- ACME credentials are present in the environment or `/etc/syscert/secrets`.

## Procedure

**1. (Optional) Test against staging first.**

```sh
sudo -u syscert syscert renew --force --staging --env-file /etc/syscert/secrets
```

This places an order against the CA's staging environment (Let's Encrypt only) and confirms
the ACME round-trip succeeds without consuming production rate limits. Skip to step 2 when
confident.

**2. Force a new ACME order.**

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
```

`--force` bypasses the expiry check and places a new order immediately. The new certificate
and a fresh keypair are written to the store (`/var/lib/syscert/`). If `[store] archive_keep`
is set, the previous set is snapshotted first.

**3. Distribute to configured targets.**

```sh
sudo -u syscert syscert distribute
```

`renew --force` writes the store but does **not** push to the paths in your `[[distribute]]`
blocks — the separate `distribute` step does that.

## Verification

Confirm the certificate was replaced and distributed:

```sh
sudo -u syscert syscert status
```

Check the expiry date on a distributed copy:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -enddate   # adjust to your path
```

The `notAfter` date should reflect the newly issued certificate.

## Rollback / recovery

The previous certificate is overwritten in the store. If `[store] archive_keep > 0`, the
snapshot is in `archive/<UTC-timestamp>/` inside the store and can be copied back manually.
Otherwise, re-run `renew --force` if the new certificate has a problem, or restore from your
own backup.

## Related procedures

- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/) — when you also need to change SANs or the key type.
- [SC-OPS-004 — Rotate the private key](/docs/procedures/rotate-key/) — explicit key-rotation notes.
- [SC-OPS-005 — Revoke and replace](/docs/procedures/revoke-and-replace/) — when the old certificate must be revoked first.

**Explanatory docs:** [Configuration](/docs/configuration/) · [Distributing certs](/docs/distributing/) ·
[Troubleshooting](/docs/troubleshooting/)
