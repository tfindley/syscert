---
title: Smallstep step-ca · DNS-01
navLabel: step-ca · DNS-01
description: A syscert.toml for an internal certificate from a Smallstep step-ca ACME provisioner via DNS-01.
order: 6
eyebrow: "// docs · sample configs"
lede: An internal certificate from a step-ca ACME provisioner. step-ca does all three challenges; this config picks dns-01, so nothing needs :80 or :443 open.
---

```toml
[cert]
hostname = "app01.internal.lan"
key_type = "ec256"

[acme]
ca            = "custom"
directory_url = "https://ca.example.com:9000/acme/acme/directory"
email         = "ops@example.com"
challenge     = "dns-01"

[acme.dns]
provider = "cloudflare"            # creds via env: CLOUDFLARE_DNS_API_TOKEN
```

The `directory_url` follows `https://<ca-host>:9000/acme/<provisioner>/directory`. Using dns-01
keeps every inbound port closed, which pairs well with an internal CA and internal DNS. If the
provisioner sets `requireEAB`, add `[acme.eab].kid` and `SYSCERT_EAB_HMAC`. Trust bootstraps the
same way as Vault: point `ca_bundle` at the CA, then run `sudo syscert trust install`.

File: [`stepca-dns-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/stepca-dns-01.toml)

The `[store]`, `[[distribute]]` and `[logging]` tail is identical in every example; see
[Sample configs](/docs/examples/).
