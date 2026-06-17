---
title: HashiCorp Vault · DNS-01
navLabel: Vault · DNS-01
description: A syscert.toml for an internal certificate from HashiCorp Vault via DNS-01 — role-scoped directory and EAB, no inbound ports.
order: 5
eyebrow: "// docs · sample configs"
lede: Same internal CA, validated by a DNS TXT record instead — so no inbound ports. This example also shows a role-scoped directory and EAB.
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

- **Validation flips direction:** with dns-01, **Vault's own resolver** queries
  `_acme-challenge.<fqdn>` — so the TXT your DNS provider publishes must be visible
  to Vault (mind split-horizon / internal DNS).
- **Role-scoped directory** (`.../roles/web/acme/directory`) ties issuance to the
  `web` role; the mount-wide form is `.../pki/acme/directory`.
- **EAB:** mint with `vault write -f pki/roles/web/acme/new-eab` — `id` is the
  `kid`, `key` is `SYSCERT_EAB_HMAC` (verbatim). Omit `[acme.eab]` if the mount's
  `eab_policy` doesn't require it. Full how-to: [EAB → Vault](/docs/eab/vault/).
- Requires a Vault version whose PKI ACME exposes `dns-01`.
- File: [`vault-dns-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-dns-01.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
