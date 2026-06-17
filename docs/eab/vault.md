---
title: EAB with HashiCorp Vault
navLabel: Vault
description: Request, list, and revoke External Account Binding tokens from Vault's PKI ACME — via the CLI or API, since the web UI doesn't expose them.
order: 1
eyebrow: "// docs · eab"
lede: Vault gates ACME registration with eab_policy. Here's how to mint, inspect, and revoke EAB tokens for SysCert — there's no web-UI form for this, so it's CLI or API.
---

Vault's PKI ACME requires EAB when the mount's `eab_policy` is `new-account-required` or
`always-required`. SysCert resolves its existing account after first registration, so **either policy
works for unattended renewal** — the token is only needed (and consumed) the first time. See
[EAB](/docs/eab/) for how SysCert consumes the `kid` + `key`.

## Request a token

EAB tokens are **bound to the exact ACME directory** they're minted under, so request from the same
mount/role your `directory_url` points at. For a directory at
`…/pki_dcauth/roles/web/acme/directory`:

```sh
vault write -f -format=json pki_dcauth/roles/web/acme/new-eab | jq -r '.data.id, .data.key'
```

The first line is the **`id`** (→ `[acme.eab].kid`); the second is the **`key`** (→
`SYSCERT_EAB_HMAC`). Issuer- and mount-wide variants exist too:
`…/issuer/:ref/acme/new-eab`, `…/acme/new-eab`.

> **Use the `key` exactly as returned — including the leading `vault-eab-0-`.** Vault's key is the
> base64url encoding of an internal marker **plus** 32 random bytes, and the ACME client decodes the
> whole string straight back to those bytes. Don't strip the prefix, truncate, or re-encode it.

## List unused tokens

Unused tokens live at the **mount** level (consumed ones disappear):

```sh
vault list pki_dcauth/eab
```

## Revoke a token

```sh
vault delete pki_dcauth/eab/<key_id>
```

## Troubleshooting: `500 … go-jose: error in cryptographic primitive`

Vault returns this when it can't verify the EAB signature — the HMAC it checks with doesn't match the
one SysCert signed with. Almost always one of:

- a **stale or already-used** token (mint a fresh one — they're single-use),
- a `kid` paired with a **different mint's** `key`,
- the `key` **mangled** (truncated, prefix stripped, re-encoded), or
- minted under a **different directory** than your `directory_url`.

Re-mint, copy `id` + `key` verbatim from the *same* response, and confirm the `new-eab` path matches
your `directory_url`. There's no web-UI form for any of this — use the CLI/API above (or the UI's
interactive console).
