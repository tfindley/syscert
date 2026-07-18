---
title: Let's Encrypt · TLS-ALPN-01
navLabel: Let's Encrypt · TLS-ALPN-01
description: A syscert.toml for a public Let's Encrypt certificate validated over port 443 (TLS-ALPN-01), with no DNS provider and no port 80.
order: 3
eyebrow: "// docs · sample configs"
lede: The modern :443-only challenge (RFC 8737). No DNS provider, and nothing on port 80.
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

Your host has to be reachable on port 443. Same `CAP_NET_BIND_SERVICE` and firewall dance as
HTTP-01, only for :443 / `https` this time.

File: [`letsencrypt-tls-alpn-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-tls-alpn-01.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see
[Sample configs](/docs/examples/).
