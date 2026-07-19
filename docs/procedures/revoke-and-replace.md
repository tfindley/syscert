---
title: "SC-OPS-005: Revoke and replace (void)"
navLabel: "005 · Revoke and replace"
description: How to revoke the current certificate at the CA and reissue plus distribute a replacement in a single run, using the void subcommand.
order: 5
eyebrow: "// docs · procedures · SC-OPS-005"
lede: void revokes the current certificate at the CA, reissues a replacement, and distributes it, all in one run. It's interactive by default; --force skips the confirmation.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-005 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `syscert` service user |
| **Last reviewed** | 2026-06-22 |

## Purpose

Revoke the current certificate at the CA (it goes invalid in OCSP/CRL), then reissue and distribute a replacement in the same run. You'd do this if the private key might be compromised. Also when a CA security event forces revocation, or when you simply need every client to stop accepting the old certificate.

## Scope

Covers the `void` subcommand: one command that revokes, reissues, then distributes. The replacement is built from your current configuration with a fresh keypair.

Does **not** cover:

- Renewal without revocation — see [SC-OPS-003](/docs/procedures/force-renewal/).
- Wiping all cert state without reissuing — see [SC-OPS-011](/docs/procedures/recover/).

## Prerequisites

- syscert is installed and a current certificate exists in the store.
- ACME credentials are present in the environment or `/etc/syscert/secrets`.
- The CA must be reachable for both the revocation and the new ACME order.

## Procedure

**1. (Optional) Test the CA connection with a staging renewal.**

For Let's Encrypt, you can sanity-check the ACME round-trip before the live revoke:

```sh
sudo -u syscert syscert renew --force --staging --env-file /etc/syscert/secrets
```

This revokes **nothing**; it's a staging renewal and that's all. Skip it if you're confident in the connection.

**2. Revoke, reissue, and distribute.**

Interactive (recommended, so you confirm before anything gets revoked):

```sh
sudo -u syscert syscert void --env-file /etc/syscert/secrets
```

You will be prompted:

```
Revoke and reissue host.example.com? The current certificate will be invalidated. [y/N]
```

Enter `y` to proceed. To skip the prompt (automation or scripts):

```sh
sudo -u syscert syscert void --force --env-file /etc/syscert/secrets
```

`void` runs three steps in order: revoke the current cert at the CA → issue a new ACME order → distribute the new certificate to every `[[distribute]]` target.

**Partial-failure behaviour.** Say the revocation request fails because the CA is briefly unreachable. `void` prints a warning and keeps going: it still reissues and distributes the replacement. The exit code comes back non-zero (`1`) when that happens, so check stderr. And if you saw the warning, the old certificate was **not** revoked.

## Verification

Confirm the new certificate is in place:

```sh
sudo -u syscert syscert status
```

Check the serial or `notBefore` on the distributed certificate to confirm it's the freshly issued one:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -serial -dates   # adjust to your path
```

To confirm the old certificate is actually revoked (assuming revocation went through), check your CA's OCSP or CRL endpoint. How you do that depends on the CA.

## Rollback / recovery

There's no automated rollback after `void`. The old certificate is already revoked at the CA, and the store now holds the replacement. If something's wrong with the new certificate, run `renew --force` for another one:

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
sudo -u syscert syscert distribute
```

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/) — renew without revoking.
- [SC-OPS-004 — Rotate the private key](/docs/procedures/rotate-key/) — key rotation without revocation.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/) — wipe and re-provision from scratch.

**Explanatory docs:** [Distributing certs](/docs/distributing/) · [Troubleshooting](/docs/troubleshooting/)
