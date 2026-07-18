---
title: Sample configurations
navLabel: Sample configs
description: Ready-to-edit syscert.toml examples for each CA and challenge — Let's Encrypt (DNS-01, HTTP-01, TLS-ALPN-01), HashiCorp Vault, and Smallstep step-ca — each on its own page.
order: 4
eyebrow: "// docs · sample configs"
lede: A starter for every CA and challenge. Each page shows only the part that differs — the [store], [[distribute]], and [logging] blocks are the same across all of them.
---

Every example lives in
[`examples/`](https://github.com/tfindley/syscert/tree/main/examples) and is
ready to copy to `/etc/syscert/syscert.toml` and edit. The full annotated
reference is
[`full.toml`](https://github.com/tfindley/syscert/blob/main/examples/full.toml);
it documents every option. The deliver/store/logging tail below is the same in
all of them:

```toml
[store]
path = "/var/lib/syscert"

[[distribute]]
artifact = "fullchain"
path     = "/etc/nginx/tls/fullchain.pem"
owner    = "root"
group    = "root"
mode     = "0644"

[[distribute]]
artifact = "privkey"
path     = "/etc/nginx/tls/privkey.pem"
owner    = "root"
group    = "root"
mode     = "0600"          # key-bearing → not world-readable

[logging]
level  = "info"
format = "text"
```

What changes between setups is the **CA** and the **challenge**. Pick the one that
matches how the CA can reach your host, or can't:

- [**Let's Encrypt · DNS-01**](/docs/examples/letsencrypt-dns-01/) — public cert via
  a DNS TXT record; no inbound ports. The most internal-friendly public option.
- [**Let's Encrypt · HTTP-01**](/docs/examples/letsencrypt-http-01/) — simplest
  public setup; no DNS provider, but needs inbound :80.
- [**Let's Encrypt · TLS-ALPN-01**](/docs/examples/letsencrypt-tls-alpn-01/) —
  modern :443-only challenge; no DNS, no :80.
- [**HashiCorp Vault · HTTP-01**](/docs/examples/vault-http-01/) — internal CA from
  Vault's PKI ACME, validated over :80.
- [**HashiCorp Vault · DNS-01**](/docs/examples/vault-dns-01/) — internal CA via
  DNS-01; role-scoped directory + EAB, no inbound ports.
- [**Smallstep step-ca · DNS-01**](/docs/examples/stepca-dns-01/) — internal CA from
  a step-ca provisioner; step-ca supports all three challenges.

Check any of these offline before you issue anything:
`sudo -u syscert syscert dry-run --config-only --config ./syscert.toml`.

Next: [Configuration reference](/docs/configuration/) ·
[Quick start](/docs/quick-start/)
