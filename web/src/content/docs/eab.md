---
title: External Account Binding (EAB)
navLabel: EAB
description: When SysCert needs an EAB key id + HMAC, how it consumes them, and how accounts, certificates, and tokens interact across runs.
order: 9
eyebrow: "// docs · eab"
lede: Some CAs gate who may register an ACME account. EAB is the standard way to prove you're allowed — here's how SysCert uses it, and how it interacts with accounts, renewals, and dry-run.
---

**External Account Binding (EAB)** authenticates ACME **account registration**: the CA issues a
**key id** (`kid`) + an **HMAC key** out of band, and the client proves possession when it registers.
SysCert needs it whenever the CA requires it — HashiCorp Vault (`eab_policy`), Smallstep step-ca
(`requireEAB`), and public CAs like ZeroSSL / Google / SSL.com.

## Configuring it

```toml
[acme.eab]
kid = "kid-from-your-ca"   # an identifier, not a secret — fine in the TOML
```

```sh
# /etc/syscert/secrets   (0640 — never in the .toml, never logged)
SYSCERT_EAB_HMAC=<the HMAC key the CA gave you, verbatim>
```

Setting `kid` enables EAB; the HMAC is read from the environment. See
[Configuration → `acme.eab`](/docs/configuration/#acmeeab--external-account-binding). For Vault
specifically — requesting, listing, and revoking tokens — see [EAB with Vault](/docs/eab/vault/).

## Accounts, certificates & dry-run

SysCert keeps **one persistent ACME account per CA** under `<store>/accounts/`, with the current
**certificate** as a child of it. That relationship is what makes EAB painless after setup:

- **EAB is sent only on first registration.** On later runs SysCert *resolves* the existing account
  (it doesn't re-register), so the EAB token is **never replayed** — renewals keep working even if the
  token was single-use or has expired, and even under Vault's `always-required` policy.
- **Full `dry-run` uses a throwaway account**, so each run registers afresh and **consumes one EAB
  token**. Use `dry-run --config-only` for an offline check that needs no token.
- **`destroy --keep-account`** drops the certificate but keeps the account, so you can force a clean
  reissue with **no new token**. Plain **`destroy`** wipes the account too, so the next run needs a
  fresh token.
- **`syscert status`** shows whether an account exists and whether `kid` is set (never the HMAC),
  alongside the certificate's issue/expiry/renewal dates.

> EAB tokens are typically **single-use and short-lived**, and are bound to the exact ACME directory
> they were minted under. Mint the `kid` + `key` together, from the directory you'll use, right before
> first issuance.
