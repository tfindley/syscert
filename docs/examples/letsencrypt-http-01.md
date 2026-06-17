---
title: Let's Encrypt · HTTP-01
navLabel: Let's Encrypt · HTTP-01
description: A syscert.toml for a public Let's Encrypt certificate validated over port 80 (HTTP-01) — no DNS provider needed.
order: 2
eyebrow: "// docs · sample configs"
lede: Simplest public setup — no DNS provider. The CA reaches the host on :80 to verify a token.
---

```toml
[cert]
hostname = "host.example.com"
key_type = "ec256"

[acme]
ca        = "letsencrypt"
email     = "you@example.com"
challenge = "http-01"           # CA validates over :80
```

- **Needs:** the host publicly reachable on **port 80**.
- Binding :80 as the unprivileged `syscert` user needs `CAP_NET_BIND_SERVICE` —
  add it to the unit's `AmbientCapabilities`/`CapabilityBoundingSet`, and open the
  firewall (RHEL: `firewall-cmd --add-service=http --permanent && --reload`).
- File: [`letsencrypt-http-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-http-01.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
