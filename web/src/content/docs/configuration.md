---
title: Configuration reference
navLabel: Configuration
description: The full syscert.toml reference — [cert], [acme], [acme.dns], [acme.eab], [store], [bundle], [[distribute]], [renewal] and [logging], plus secrets handling and CA support.
order: 3
eyebrow: "// docs · configuration"
lede: Everything syscert reads from syscert.toml, section by section. Secrets are the one thing that never live here.
---

Configuration is TOML. The default location is `/etc/syscert/syscert.toml`;
override it with `--config <path>` or the `SYSCERT_CONFIG` environment variable
(precedence: `--config` flag → `SYSCERT_CONFIG` → default). The shipped systemd
unit reads `SYSCERT_CONFIG` from `/etc/default/syscert`, so you can point the
service at a different file once, with no unit edit.

Ready-to-edit files live in the repo's
[examples/](https://github.com/tfindley/syscert/tree/main/examples) — a
fully-commented
[full.toml](https://github.com/tfindley/syscert/blob/main/examples/full.toml)
covering every option, plus focused starters for Let's Encrypt, Vault and step-ca.

> **Secrets never go in this file.** DNS-provider tokens, CA credentials and the
> EAB HMAC are read from the environment (typically `/etc/syscert/secrets`, mode
> `0640`) and are never logged. See [`[acme.dns]`](#acmedns--dns-provider--credentials).

## `[cert]` — certificate subject

| Key | Default | Description |
|---|---|---|
| `hostname` | system FQDN | The name the cert is built around. If empty, syscert uses the host's FQDN; **if the host has no FQDN it errors and refuses to run** (it never guesses). |
| `sans` | `[]` | Extra DNS Subject Alternative Names. |
| `ip_sans` | `[]` | IP SANs. Setting this **forces the challenge to http-01/tls-alpn-01** (RFC 8738 forbids DNS-01 for IPs) and the CA must reach the host on :80/:443. Private (RFC 1918) IPs require an internal CA — a public CA is rejected up front. |
| `key_type` | `ec256` | `ec256` · `ec384` · `rsa2048` · `rsa4096`. A **fresh keypair is generated each renewal**. |
| `reuse_key` | `false` | Keep the same keypair across renewals — only needed if a consumer pins the public key. |

```toml
[cert]
hostname = "host.example.com"
sans     = ["www.example.com"]
key_type = "ec256"
```

## `[acme]` — CA and challenge

| Key | Default | Description |
|---|---|---|
| `ca` | *required* | `letsencrypt` (public CA — built-in directory URLs + `--staging`) · `custom` (any internal/other ACME CA: Vault, step-ca, … set via `directory_url`). |
| `directory_url` | per-CA | The ACME directory endpoint URL. **Required** when `ca = "custom"`; for `letsencrypt` it defaults to production. |
| `email` | *required* | ACME account contact address. |
| `challenge` | `dns-01` | `dns-01` · `http-01` · `tls-alpn-01` · `dns-persist-01`. Auto-switched to http-01/tls-alpn-01 when `ip_sans` is set; `dns-persist-01` is opt-in and capability-checked at runtime. **http-01/tls-alpn-01 need the CA to reach this host on inbound :80/:443**; dns-01 needs no inbound ports. |
| `profile` | `""` | ACME profile to request (e.g. `shortlived` → ~6-day certs, required for public-CA IP certs). Validated at runtime against the directory's `meta.profiles`. |
| `ca_bundle` | `""` | Path to a PEM of the internal CA to trust **for the ACME connection only** (not the system store). Bootstraps issuance against a Vault/step-ca the host doesn't trust yet; syscert warns when it's set. See [Troubleshooting](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca). |

### Which directory_url for your CA

| CA                    | `ca`          | `directory_url`                                                                                                 |
| --------------------- | ------------- | --------------------------------------------------------------------------------------------------------------- |
| Let's Encrypt         | `letsencrypt` | *leave empty* → production. Set staging `https://acme-staging-v02.api.letsencrypt.org/directory` while testing. |
| HashiCorp Vault PKI   | `custom`      | `https://<vault>:8200/v1/<mount>/acme/directory`. Requires ACME enabled on the mount.                           |
| Smallstep step-ca     | `custom`      | `https://<ca-host>:9000/acme/<provisioner>/directory`.                                                          |
| Any other ACME server | `custom`      | its RFC 8555 directory URL.                                                                                     |

> **All three CAs support `dns-01`, `http-01`, and `tls-alpn-01`.** Vault's PKI
> ACME exposes `dns-01` in current versions (older releases had `http-01` /
> `tls-alpn-01` only — confirm your Vault version).

```toml
# Let's Encrypt (production) via DNS-01
[acme]
ca        = "letsencrypt"
email     = "you@example.com"
challenge = "dns-01"

# HashiCorp Vault (internal CA) via HTTP-01
[acme]
ca            = "custom"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "you@example.com"
challenge     = "http-01"
```

## `[acme.dns]` — DNS provider + credentials

Used only when `challenge` is `dns-01` or `dns-persist-01`.

| Key | Default | Description |
|---|---|---|
| `provider` | `""` | Any [lego DNS-provider id](https://go-acme.github.io/lego/dns/) (e.g. `cloudflare`, `gandiv5`, `route53`). |
| `propagation_check` | `all` | `all` — visible on the local resolver *and* the authoritative NS (lego default). `authoritative` — verify only on the CA's authoritative NS; skip the local check (use on split-horizon/VPN/slow resolvers). `off` — skip the local pre-check entirely. |

Credentials are supplied via the environment (or a restricted secrets file),
**never in the config**. Each lego provider reads its own variables — e.g.
`CLOUDFLARE_DNS_API_TOKEN` (cloudflare), `GANDIV5_PERSONAL_ACCESS_TOKEN` (Gandi
LiveDNS), or `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_REGION`
(route53). See the [lego provider docs](https://go-acme.github.io/lego/dns/) for
the exact names. The systemd unit loads these from `/etc/syscert/secrets`; for a
manual run, pass `--env-file /etc/syscert/secrets` (repeatable; the existing
environment wins) instead of exporting each one.

```toml
[acme.dns]
provider          = "gandiv5"
propagation_check = "authoritative"   # only if the local resolver is slow to see public DNS
```

## `[acme.eab]` — External Account Binding

Some CAs require **External Account Binding** to register an ACME account: they
issue a Key ID + HMAC key out-of-band and the client proves possession at
registration. Used by Vault (`eab_policy`), step-ca (`requireEAB`), and public CAs
like ZeroSSL / Google / SSL.com.

| Key | Default | Description |
|---|---|---|
| `kid` | `""` | EAB Key ID. Setting it **enables EAB**. An identifier, not a secret — fine in this file. |

The **HMAC key is a secret** — supply it via `SYSCERT_EAB_HMAC` (the base64url key
the CA gave you) in `/etc/syscert/secrets`, never in the TOML and never logged. EAB
is checked by the CA only when the account is first created; syscert reuses its
persistent account key afterwards. Requesting one from HashiCorp Vault?
See [EAB](/docs/eab/) — and [EAB → Vault](/docs/eab/vault/) to request one.

```toml
[acme.eab]
kid = "kid-from-your-ca"          # + export SYSCERT_EAB_HMAC=<base64url-hmac> in /etc/syscert/secrets
```

## `[store]` — canonical store

| Key | Default | Description |
|---|---|---|
| `path` | `/var/lib/syscert` | Where syscert keeps the source-of-truth cert material and ACME account state. Owned by the `syscert` user; key-bearing files kept `0600`. |
| `dir_mode` | `0700` | Octal mode for the store directory. Widen it (e.g. `0750`) together with `group` to let a group traverse the store and read the public artifacts (`cert`/`chain`/`fullchain`); **private keys stay `0600`** either way. |
| `group` | `syscert` | Group that owns the store dir and files. Set it (with a `dir_mode` that grants group access) to let a trusted group read the store directly. |
| `archive_keep` | `0` | Keep this many previous certificate sets. `0` overwrites in place (no history). When >0, each renewal first snapshots the current artifacts to `archive/<UTC-timestamp>/` inside the store (real files, no symlinks), pruned to the newest N. |

> Prefer **[[distribute]]** for giving a specific service its cert — it copies a real file
> with that consumer's exact owner/mode (including the key). `group`/`dir_mode` only widen
> direct read of the *public* artifacts; they never expose private keys. Archived snapshots
> live inside the locked store and are never distributed, so consumers are unaffected.

## `[bundle]` — all-in-one file

Controls the composition of `bundle.pem`.

| Key | Default | Description |
|---|---|---|
| `order` | `["cert","chain","root","key"]` | Components and their order. **Omit a token to exclude it.** Tokens: `cert` (leaf), `chain` (intermediates), `root`, `key`. |

```toml
[bundle]
order = ["key", "cert", "chain"]   # key first, no root
```

The `root` is dropped automatically when the CA provides none (public CAs). If
`key` is present, any target receiving `bundle` must use a non-world-readable mode.

## `[[distribute]]` — delivering to consumers

Zero or more blocks; each copies **one artifact** to a path with the
ownership/mode/context that consumer needs. syscert overwrites only the paths it
manages and **does not reload consumers** — see
[Distributing certs](/docs/distributing/).

| Key | Description |
|---|---|
| `artifact` *(required)* | Which file to place: `cert` · `privkey` · `chain` · `fullchain` · `bundle`. |
| `path` *(required)* | Destination path. |
| `owner` / `group` | File owner and group. |
| `mode` | Octal mode, e.g. `"0644"`. **privkey/bundle hold the key — a world-readable mode is rejected.** |
| `selinux_context` | Optional SELinux file context (RHEL family), e.g. `cert_t`. |

```toml
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
mode     = "0600"
```

## `[renewal]` / `[logging]`

| Key | Default | Description |
|---|---|---|
| `renewal.renew_before` | `""` (auto) | Empty = derive the window from the cert's lifetime (short-lived certs renew ~daily; long-lived use a wide window). Set a duration like `"30d"` to override. |
| `logging.level` | `info` | `debug` · `info` · `warn` · `error`. |
| `logging.format` | `text` | `text` (journald-friendly) · `json`. |

Operational logs (events, errors, and lego's ACME output) go to **stderr**;
command results and prompts go to **stdout**. Secret values are never logged.

---

Next: [Distributing certs](/docs/distributing/) ·
[Troubleshooting](/docs/troubleshooting/) ·
[full.toml on GitHub](https://github.com/tfindley/syscert/blob/main/examples/full.toml)
