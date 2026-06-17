---
title: Smallstep step-ca · DNS-01
navLabel: step-ca · DNS-01
description: A syscert.toml for an internal certificate from a Smallstep step-ca ACME provisioner via DNS-01.
order: 6
eyebrow: "// docs · sample configs"
lede: Internal certificate from a step-ca ACME provisioner. step-ca supports all three challenges; this one uses dns-01, so no inbound :80/:443.
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

- `directory_url` format: `https://<ca-host>:9000/acme/<provisioner>/directory`.
- DNS-01 means no inbound ports — combine an internal CA with internal DNS.
- If the provisioner has `requireEAB`, set `[acme.eab].kid` + `SYSCERT_EAB_HMAC`.
- Bootstrap trust with `ca_bundle`, then `sudo syscert trust install`, the same as
  Vault.
- File: [`stepca-dns-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/stepca-dns-01.toml)

The shared `[store]` / `[[distribute]]` / `[logging]` tail is identical across every
example — see [Sample configs](/docs/examples/).
