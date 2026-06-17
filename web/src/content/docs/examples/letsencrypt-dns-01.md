---
title: Let's Encrypt · DNS-01
navLabel: Let's Encrypt · DNS-01
description: A syscert.toml for a public Let's Encrypt certificate validated by a DNS-01 TXT record — no inbound ports required.
order: 1
eyebrow: "// docs · sample configs"
lede: Public certificate validated by a DNS TXT record — works even with no inbound ports. The most internal-friendly public option.
---

```toml
[cert]
hostname = "host.example.com"
key_type = "ec256"

[acme]
ca        = "letsencrypt"
email     = "you@example.com"
challenge = "dns-01"

[acme.dns]
provider = "gandiv5"            # creds via env: GANDIV5_PERSONAL_ACCESS_TOKEN
```

- **Needs:** a DNS provider [lego supports](https://go-acme.github.io/lego/dns/)
  and its API token in the environment (e.g. `GANDIV5_PERSONAL_ACCESS_TOKEN`).
- **No inbound :80/:443** required — good for hosts behind NAT or a firewall.
- Set `propagation_check = "authoritative"` if the host's resolver is
  split-horizon / on a VPN / slow to see public DNS.
- File: [`letsencrypt-dns-01-gandiv5.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-dns-01-gandiv5.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
