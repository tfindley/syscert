---
title: HashiCorp Vault · HTTP-01
navLabel: Vault · HTTP-01
description: A syscert.toml for an internal certificate from HashiCorp Vault's PKI ACME endpoint, validated over port 80 (HTTP-01).
order: 4
eyebrow: "// docs · sample configs"
lede: An internal certificate from Vault's PKI ACME endpoint. What flips it from the public examples is ca = "custom" plus a directory_url.
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

Vault handles all three challenges; this config uses http-01, where Vault reaches the host on
:80. If you'd rather validate over DNS, see [Vault · DNS-01](/docs/examples/vault-dns-01/).

Put an IPv4 address in the directory URL, since Vault has an IPv6-ACME quirk. The mount also has
to have ACME turned on (`vault write pki/config/acme enabled=true` plus a `pki/config/cluster`).

If the host doesn't trust Vault's CA yet, set `ca_bundle` to bootstrap the ACME connection, then
run `sudo syscert trust install` for host-wide trust. The
[Troubleshooting](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca) page
covers the x509 error you'll hit otherwise. And if Vault wants EAB, set `[acme.eab].kid` and
`SYSCERT_EAB_HMAC` in the environment; the [EAB → Vault](/docs/eab/vault/) guide walks through it.

File: [`vault-http-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-http-01.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see
[Sample configs](/docs/examples/).
