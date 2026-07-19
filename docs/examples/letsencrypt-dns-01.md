---
title: Let's Encrypt · DNS-01
navLabel: Let's Encrypt · DNS-01
description: A syscert.toml for a public Let's Encrypt certificate proven by a DNS-01 TXT record, so no inbound ports are needed.
order: 1
eyebrow: "// docs · sample configs"
lede: A public certificate proven by a DNS TXT record. Works even when the host has no inbound ports open, which makes it the friendliest public option for locked-down machines.
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

You need a DNS provider that [lego supports](https://go-acme.github.io/lego/dns/), plus its API token in the environment (for Gandi that's `GANDIV5_PERSONAL_ACCESS_TOKEN`). Nothing has to be reachable on :80 or :443, so this suits hosts stuck behind NAT or a firewall. If the host's resolver is split-horizon, sits on a VPN, or is just slow to see public DNS, set `propagation_check = "authoritative"`.

File: [`letsencrypt-dns-01-gandiv5.toml`](https://github.com/tfindley/syscert/blob/main/examples/letsencrypt-dns-01-gandiv5.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see [Sample configs](/docs/examples/).
