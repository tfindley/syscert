---
title: Let's Encrypt · HTTP-01
navLabel: Let's Encrypt · HTTP-01
description: A syscert.toml for a public Let's Encrypt certificate validated over port 80 (HTTP-01), with no DNS provider to set up.
order: 2
eyebrow: "// docs · sample configs"
lede: The simplest public setup, with no DNS provider to wire up. The CA reaches your host on :80 and checks for a token.
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

Your host has to be reachable from the public internet on port 80. syscert runs as an unprivileged user, and binding :80 as that `syscert` user needs `CAP_NET_BIND_SERVICE`, so add it to the unit's `AmbientCapabilities`/`CapabilityBoundingSet` and open the firewall (on RHEL: `firewall-cmd --add-service=http --permanent && --reload`).

File: [`letsencrypt-http-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-http-01.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see [Sample configs](/docs/examples/).
