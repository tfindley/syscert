---
title: "SC-OPS-005: Revoke and replace (void)"
navLabel: "005 · Revoke and replace"
description: Formal procedure for revoking the current certificate at the CA and immediately reissuing and distributing a replacement — using the void subcommand.
order: 5
eyebrow: "// docs · procedures · SC-OPS-005"
lede: void revokes the current certificate at the CA, reissues a replacement, and distributes it in one step. Interactive by default; --force skips the confirmation.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-005 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `syscert` service user |
| **Last reviewed** | 2026-06-22 |

## Purpose

Revoke the current certificate at the CA (marking it invalid in OCSP/CRL), then immediately
reissue and distribute a replacement. Use this when the private key is suspected compromised,
when a CA security event requires revocation, or when you need to guarantee clients will not
accept the old certificate.

## Scope

Covers the `void` subcommand, which revokes + reissues + distributes in a single step. The
replacement certificate uses the current configuration with a fresh keypair.

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

This does **not** revoke anything — it is a staging renewal only. Skip if you are confident.

**2. Revoke, reissue, and distribute.**

Interactive (recommended — confirm before revoking):

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

`void` performs three steps in sequence: revoke the current cert at the CA → issue a new ACME
order → distribute the new certificate to all `[[distribute]]` targets.

**Partial-failure behaviour.** If the revocation request fails (e.g. the CA is temporarily
unreachable), `void` prints a warning and continues — it still reissues and distributes the
replacement. The exit code will be non-zero (`1`) in that case; check stderr. The old
certificate was **not** revoked if the warning appeared.

## Verification

Confirm the new certificate is in place:

```sh
sudo -u syscert syscert status
```

Check the serial number or `notBefore` of the distributed certificate to confirm it is the
newly issued one:

```sh
openssl x509 -in /etc/nginx/tls/fullchain.pem -noout -serial -dates   # adjust to your path
```

To verify the old certificate is revoked (if revocation succeeded), use your CA's OCSP or
CRL endpoint — the mechanism is CA-specific.

## Rollback / recovery

There is no automated rollback after `void` — the old certificate has been revoked at the CA
and the store holds the replacement. If the new certificate has a problem, run
`renew --force` to get another replacement:

```sh
sudo -u syscert syscert renew --force --env-file /etc/syscert/secrets
sudo -u syscert syscert distribute
```

## Related procedures

- [SC-OPS-003 — Force an immediate renewal](/docs/procedures/force-renewal/) — renew without revoking.
- [SC-OPS-004 — Rotate the private key](/docs/procedures/rotate-key/) — key rotation without revocation.
- [SC-OPS-011 — Recover from a broken state](/docs/procedures/recover/) — wipe and re-provision from scratch.

**Explanatory docs:** [Distributing certs](/docs/distributing/) · [Troubleshooting](/docs/troubleshooting/)
