---
title: HashiCorp Vault · HTTP-01
navLabel: Vault · HTTP-01
description: A syscert.toml for an internal certificate from HashiCorp Vault's PKI ACME endpoint, validated over port 80 (HTTP-01).
order: 4
eyebrow: "// docs · sample configs"
lede: Internal certificate from Vault's PKI ACME endpoint. The defining change is ca = "custom" plus a directory_url.
---

```toml
[cert]
hostname = "app01.internal.lan"
key_type = "ec256"

[acme]
ca            = "custom"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "ops@example.com"
challenge     = "http-01"          # Vault also supports dns-01 / tls-alpn-01

# bootstrap trust if the host doesn't trust Vault's CA yet, then `trust install`:
# ca_bundle   = "/etc/syscert/vault-ca.pem"
```

- **Vault supports all three challenges** — this one uses http-01 (Vault reaches
  the host on :80); for dns-01 see [Vault · DNS-01](/docs/examples/vault-dns-01/).
- Use **IPv4** in the directory URL (Vault has an IPv6-ACME quirk). The mount needs
  ACME enabled (`vault write pki/config/acme enabled=true` + a `pki/config/cluster`).
- If the host doesn't trust Vault's CA yet, set `ca_bundle` to bootstrap the ACME
  connection, then `sudo syscert trust install` for host-wide trust — see
  [Troubleshooting](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca).
- If Vault requires EAB, set `[acme.eab].kid` + `SYSCERT_EAB_HMAC` in the env — see
  [EAB → Vault](/docs/eab/vault/).
- File: [`vault-http-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-http-01.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
