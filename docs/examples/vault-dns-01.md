---
title: HashiCorp Vault · DNS-01
navLabel: Vault · DNS-01
description: A syscert.toml for an internal certificate from HashiCorp Vault via DNS-01, showing a role-scoped directory and EAB, with no inbound ports.
order: 5
eyebrow: "// docs · sample configs"
lede: The same internal CA, but proven by a DNS TXT record this time, so no inbound ports. This one also shows a role-scoped directory and EAB.
---

```toml
[cert]
hostname = "web01.internal.lan"
key_type = "ec256"

[acme]
ca            = "custom"
# role-scoped: issuance follows the "web" role's policy
directory_url = "https://vault.example.com:8200/v1/pki/roles/web/acme/directory"
email         = "ops@example.com"
challenge     = "dns-01"

[acme.dns]
provider = "cloudflare"            # creds via env: CLOUDFLARE_DNS_API_TOKEN

[acme.eab]
kid = "kid-from-vault"             # + SYSCERT_EAB_HMAC in the env
```

With dns-01 the validation runs the other way. Vault's own resolver queries `_acme-challenge.<fqdn>`, so the TXT record your DNS provider publishes has to be visible to Vault. Watch split-horizon and internal DNS here.

The role-scoped directory (`.../roles/web/acme/directory`) pins issuance to the `web` role; drop back to `.../pki/acme/directory` for the mount-wide form.

For EAB, mint a credential with `vault write -f pki/roles/web/acme/new-eab`. The returned `id` is your `kid`, and `key` is `SYSCERT_EAB_HMAC`, copied verbatim. Skip `[acme.eab]` entirely if the mount's `eab_policy` doesn't ask for it. The full how-to is [EAB → Vault](/docs/eab/vault/). One version caveat: your Vault build needs a PKI ACME that exposes `dns-01`.

File: [`vault-dns-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-dns-01.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see [Sample configs](/docs/examples/).
