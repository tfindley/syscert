---
title: Sample configurations
navLabel: Sample configs
description: Ready-to-edit syscert.toml examples for each CA and challenge — Let's Encrypt (DNS-01, HTTP-01, TLS-ALPN-01), HashiCorp Vault, and Smallstep step-ca — with notes on what differs.
order: 3
eyebrow: "// docs · sample configs"
lede: A starter for every CA and challenge. Each shows only the part that differs — the [store], [[distribute]], and [logging] blocks are the same across all of them.
---

Every example lives in
[`examples/`](https://github.com/tfindley/syscert/tree/main/examples) and is
ready to copy to `/etc/syscert/syscert.toml` and edit. The full annotated
reference is
[`full.toml`](https://github.com/tfindley/syscert/blob/main/examples/full.toml) —
it documents every option. The deliver/store/logging tail below is identical in
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
matches how the CA can reach (or not reach) your host.

## Let's Encrypt · DNS-01 (Gandi)

Public certificate validated by a DNS TXT record — works even with **no inbound
ports**. The most internal-friendly public option.

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

## Let's Encrypt · HTTP-01

Simplest public setup — **no DNS provider**. The CA reaches the host on **:80** to
verify a token.

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

## Let's Encrypt · TLS-ALPN-01

Modern **:443-only** challenge (RFC 8737) — no DNS provider and no port 80.

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

## HashiCorp Vault · HTTP-01

Internal certificate from Vault's PKI ACME endpoint. The defining change is
`ca = "custom"` plus a `directory_url`.

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

- **Vault supports all three challenges** — this one uses http-01 (Vault reaches
  the host on :80); for dns-01 see the next example.
- Use **IPv4** in the directory URL (Vault has an IPv6-ACME quirk). The mount needs
  ACME enabled (`vault write pki/config/acme enabled=true` + a `pki/config/cluster`).
- If the host doesn't trust Vault's CA yet, set `ca_bundle` to bootstrap the ACME
  connection, then `sudo syscert trust install` for host-wide trust — see
  [Troubleshooting](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca).
- If Vault requires EAB, set `[acme.eab].kid` + `SYSCERT_EAB_HMAC` in the env.
- File: [`vault-http-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-http-01.toml)

## HashiCorp Vault · DNS-01

Same internal CA, validated by a **DNS TXT record** instead — so **no inbound
ports**. This example also shows a **role-scoped** directory and **EAB**.

```toml
[cert]
hostname = "web01.internal.lan"
key_type = "ec256"

[acme]
ca            = "custom"
# role-scoped: issuance follows the "web" role's policy
directory_url = "https://vault.example.com:8200/v1/pki/roles/web/acme/directory"
email         = "ops@example.com"
challenge     = "dns-01"

[acme.dns]
provider = "cloudflare"            # creds via env: CLOUDFLARE_DNS_API_TOKEN

[acme.eab]
kid = "kid-from-vault"             # + SYSCERT_EAB_HMAC in the env
```

- **Validation flips direction:** with dns-01, **Vault's own resolver** queries
  `_acme-challenge.<fqdn>` — so the TXT your DNS provider publishes must be visible
  to Vault (mind split-horizon / internal DNS).
- **Role-scoped directory** (`.../roles/web/acme/directory`) ties issuance to the
  `web` role; the mount-wide form is `.../pki/acme/directory`.
- **EAB:** mint with `vault write -f pki/roles/web/acme/new-eab` — `id` is the
  `kid`, `key` is `SYSCERT_EAB_HMAC`. Omit `[acme.eab]` if the mount's `eab_policy`
  doesn't require it.
- Requires a Vault version whose PKI ACME exposes `dns-01`.
- File: [`vault-dns-01.toml`](https://github.com/tfindley/syscert/blob/main/examples/vault-dns-01.toml)

## Smallstep step-ca · DNS-01

Internal certificate from a step-ca ACME provisioner. step-ca supports **all three
challenges**; this one uses dns-01, so **no inbound :80/:443**.

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

---

Validate any of these offline before issuing:
`sudo -u syscert syscert dry-run --config-only --config ./syscert.toml`.

Next: [Configuration reference](/docs/configuration/) ·
[Quick start](/docs/quick-start/)
