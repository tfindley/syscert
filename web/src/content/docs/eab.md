---
title: External Account Binding (EAB)
navLabel: EAB
description: When SysCert needs an EAB key id and HMAC, how it uses them, and how accounts, certificates, and tokens interact across runs.
order: 5
eyebrow: "// docs · eab"
lede: Some CAs gate who's allowed to register an ACME account. EAB is the standard way to prove you're allowed. This page covers how SysCert uses it, and how it interacts with accounts, renewals, and dry-run.
---

**External Account Binding (EAB)** authenticates ACME **account registration**. The CA issues a **key id** (`kid`) and an **HMAC key** out of band, and the client proves it holds them when it registers. SysCert needs EAB whenever the CA requires it: HashiCorp Vault (`eab_policy`), Smallstep step-ca (`requireEAB`), and public CAs like ZeroSSL, Google, or SSL.com.

## Configuring it

```toml
[acme.eab]
kid = "kid-from-your-ca"   # an identifier, not a secret — fine in the TOML
```

```sh
# /etc/syscert/secrets   (0640 — never in the .toml, never logged)
SYSCERT_EAB_HMAC=<the HMAC key the CA gave you, verbatim>
```

Setting `kid` turns EAB on; the HMAC comes from the environment. See [Configuration → `acme.eab`](/docs/configuration/#acmeeab--external-account-binding). For Vault specifically (requesting, listing, and revoking tokens), see [EAB with Vault](/docs/eab/vault/).

## Accounts, certificates & dry-run

SysCert keeps **one persistent ACME account per CA** under `<store>/accounts/`, with the current **certificate** as a child of it. That relationship is what makes EAB painless once you're set up.

EAB is sent only on that first registration. On later runs SysCert *resolves* the existing account instead of re-registering, so the token is **never replayed**. Renewals keep working even if the token was single-use or has already expired, and even under Vault's `always-required` policy.

A full `dry-run` is different. It uses a throwaway account, registers afresh every time, and **consumes one EAB token** per run. For an offline check that needs no token, use `dry-run --config-only`.

`destroy --keep-account` drops the certificate but keeps the account, so you can force a clean reissue with **no new token**. Plain `destroy` wipes the account too, and the next run then needs a fresh token.

`syscert status` tells you whether an account exists and whether `kid` is set (never the HMAC), next to the certificate's issue, expiry, and renewal dates.

> EAB tokens are usually **single-use and short-lived**, and they're bound to the exact ACME
> directory they were minted under. Mint the `kid` + `key` together, from the directory you'll
> actually use, right before first issuance.
