---
title: Let's Encrypt · TLS-ALPN-01
navLabel: Let's Encrypt · TLS-ALPN-01
description: A syscert.toml for a public Let's Encrypt certificate validated over port 443 (TLS-ALPN-01) — no DNS provider, no port 80.
order: 3
eyebrow: "// docs · sample configs"
lede: Modern :443-only challenge (RFC 8737) — no DNS provider and no port 80.
---

```toml
[cert]
hostname = "host.example.com"
key_type = "ec256"

[acme]
ca        = "letsencrypt"
email     = "you@example.com"
challenge = "tls-alpn-01"       # CA validates over :443
```

- **Needs:** the host publicly reachable on **port 443**.
- Same `CAP_NET_BIND_SERVICE` + firewall note as HTTP-01 (for :443 / `https`).
- File: [`letsencrypt-tls-alpn-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-tls-alpn-01.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
