# SysCert

A small Linux service that obtains and auto-renews a **TLS certificate for the machine's own
hostname**, then delivers it to local consumers (nginx, Cockpit, …) with the ownership, permissions
and SELinux context each one needs. It runs as a least-privilege systemd service, independent of any
system `certbot`.

It speaks ACME via [lego](https://go-acme.github.io/lego/), so it works against **Let's Encrypt**,
a **HashiCorp Vault** PKI ACME endpoint, or **Smallstep `step-ca`** — and ships certbot-compatible
output files plus an all-in-one bundle.

> **Project status: early / greenfield.** `syscert dry-run` works today: it validates the config,
> resolves the subject, and runs the **full ACME order + challenge** against the CA *without saving
> anything* — the same idea as `certbot --dry-run` (Let's Encrypt uses staging automatically). What's
> **not implemented yet:** persisting certificates to the store, distribution to consumers, the
> renewal loop, and the trust-store commands.

---

## Contents
- [Install](#install)
- [Quick start](#quick-start)
- [Commands](#commands)
- [Configuration](#configuration)
  - [`[cert]`](#cert--certificate-subject)
  - [`[acme]` and the CA / `directory_url`](#acme--ca-and-challenge)
  - [`[acme.dns]` + secrets](#acmedns--dns-provider--credentials)
  - [`[store]`](#store--canonical-store)
  - [`[bundle]`](#bundle--all-in-one-file)
  - [`[[distribute]]`](#distribute--delivering-to-consumers)
  - [`[renewal]` / `[logging]`](#renewal--logging)
- [Output files](#output-files)
- [Full example](#full-example)

---

## Install

**Supported targets:** Debian/Ubuntu and the RHEL family (others may work but aren't tested).

There are no pre-built binaries yet, so build from source.

### 1. Install Go (≥ 1.26)

```sh
# official tarball (amd64); see https://go.dev/dl for other archs / latest version
curl -fsSL https://go.dev/dl/go1.26.4.linux-amd64.tar.gz -o /tmp/go.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH=/usr/local/go/bin:$PATH      # add to your shell profile to persist
go version
```

### 2. Build

```sh
git clone https://github.com/tfindley/syscert.git
cd syscert
go build -o syscert ./cmd/syscert
./syscert --help
```

### 3. Install the binary (optional, for a real deployment)

```sh
sudo install -m 0755 syscert /usr/local/bin/syscert
sudo mkdir -p /etc/syscert            # config lives here
sudo install -m 0644 config.sample.toml /etc/syscert/syscert.toml
```

> A dedicated `syscert` system user, the `/var/lib/syscert` store, the systemd unit, and an Ansible
> role are part of the planned packaging — not wired up yet.

---

## Quick start

```sh
# copy the sample and edit it
cp config.sample.toml /etc/syscert/syscert.toml

# offline: validate config + resolve the subject only
syscert dry-run --config-only --config /etc/syscert/syscert.toml

# full dry-run: also run the real ACME order + challenge, saving nothing.
# Provider/CA credentials come from the environment (see the secrets table below),
# e.g. for the Gandi DNS-01 example:
GANDI_API_KEY=… syscert dry-run --config /etc/syscert/syscert.toml
```

A passing `--config-only` run prints the resolved subject, CA and challenge:

```
config OK:
  subject:   demo.internal.lan
  CA:        vault
  challenge: http-01
```

A failing config lists every problem with an actionable message and exits non-zero:

```
FAIL: 2 config problem(s)
  - cert.ip_sans: IP SANs require challenge http-01 or tls-alpn-01 (RFC 8738 forbids dns-01)
  - distribute[0].mode: artifact "privkey" holds the private key; mode "0644" is world-readable
```

The **full** dry-run goes on to perform the real ACME order + challenge (creating/cleaning a DNS-01
TXT record, or binding :80/:443 for http-01/tls-alpn-01) against the CA — Let's Encrypt automatically
uses **staging**. The obtained certificate is discarded, not written to disk.

---

## Commands

| Command | Status | Purpose |
|---|---|---|
| `syscert dry-run --config <path>` | **working** | Config test + the full ACME order/challenge against the CA. Nothing is saved (LE uses staging). |
| `syscert dry-run --config-only …` | **working** | Validate config + resolve the subject only; no network. |
| `syscert issue --config <path>` | planned | One-shot issuance into the store, then distribute. |
| `syscert renew --config <path>` | planned | Force a renewal now. |
| `syscert run --config <path>` | planned | Long-running issuance + renewal loop (the service). |
| `syscert void --config <path>` | planned | Revoke/discard the current cert and request a fresh one. |
| `syscert destroy --config <path>` | planned | Tear down config/state and re-provision (e.g. switch CA). |
| `syscert trust install` / `trust remove` | planned | Add/remove an internal CA's root in the system trust store (run as root). |

---

## Configuration

TOML. Default location `/etc/syscert/syscert.toml`; pass any path with `--config`.
**Secrets never go in this file** — see [DNS provider credentials](#acmedns--dns-provider--credentials).

### `[cert]` — certificate subject

| Key | Type | Default | Description |
|---|---|---|---|
| `hostname` | string | system FQDN | The name the cert is built around. If empty, SysCert uses the host's FQDN; **if the host has no FQDN it errors and refuses to run** (it never guesses). |
| `sans` | list | `[]` | Extra DNS Subject Alternative Names. |
| `ip_sans` | list | `[]` | IP SANs. Setting this **forces the challenge to `http-01`/`tls-alpn-01`** (RFC 8738 forbids DNS-01 for IPs), and the CA must reach the host on :80/:443. Private (RFC 1918) IPs require an **internal CA** — a public CA will be rejected up front. |
| `key_type` | string | `ec256` | `ec256` \| `ec384` \| `rsa2048` \| `rsa4096`. A **fresh keypair is generated each renewal**. |
| `reuse_key` | bool | `false` | Keep the same keypair across renewals — only needed if a consumer pins the public key. |

```toml
[cert]
hostname = "host.example.com"
sans     = ["www.example.com"]
key_type = "ec256"
```

### `[acme]` — CA and challenge

| Key | Type | Default | Description |
|---|---|---|---|
| `ca` | string | *(required)* | `letsencrypt` \| `vault` \| `stepca` \| `custom`. Selects known defaults and behaviour. |
| `directory_url` | string | per-CA | The ACME **directory endpoint URL** (see below). **Required** for `vault`, `stepca`, `custom`. For `letsencrypt` it defaults to production. |
| `email` | string | *(required)* | ACME account contact address. |
| `challenge` | string | `dns-01` | `dns-01` (default) \| `http-01` \| `tls-alpn-01` \| `dns-persist-01`. Auto-switched to `http-01`/`tls-alpn-01` when `ip_sans` is set. `dns-persist-01` is opt-in and capability-checked at runtime. |
| `profile` | string | `""` | ACME *profile* to request (e.g. `shortlived`). Leave empty unless you specifically need one — `shortlived` yields ~6-day certs (required for public-CA **IP** certs). Validated at runtime against the directory's `meta.profiles`. |

#### What is `directory_url`?

Every ACME CA publishes a single **directory** JSON document — the entry point a client fetches to
discover all the other endpoints (`newAccount`, `newOrder`, `newNonce`, `revokeCert`, …) and the
CA's metadata (RFC 8555 §7.1.1). `directory_url` is just that URL. Point SysCert at the right one
for your CA:

| CA (`ca =`) | `directory_url` | Notes |
|---|---|---|
| `letsencrypt` | *(leave empty)* → `https://acme-v02.api.letsencrypt.org/directory` | Set it explicitly to the **staging** URL `https://acme-staging-v02.api.letsencrypt.org/directory` while testing (staging has far higher rate limits; its certs aren't publicly trusted). |
| `vault` | `https://<vault-addr>:8200/v1/<pki-mount>/acme/directory` | e.g. `https://vault.example.com:8200/v1/pki/acme/directory`. Role/issuer-scoped variants exist: `.../v1/pki/roles/<role>/acme/directory`. Requires ACME enabled on the PKI mount (`vault write pki/config/acme enabled=true` and a `pki/config/cluster path=…`). |
| `stepca` | `https://<ca-host>:9000/acme/<provisioner>/directory` | e.g. `https://ca.example.com:9000/acme/acme/directory`. `<provisioner>` is the name of your ACME provisioner in step-ca. |
| `custom` | any RFC 8555 directory URL | For any other ACME server. |

```toml
# Let's Encrypt (production) via DNS-01
[acme]
ca        = "letsencrypt"
email     = "you@example.com"
challenge = "dns-01"

# HashiCorp Vault (internal CA) via HTTP-01
[acme]
ca            = "vault"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "you@example.com"
challenge     = "http-01"
```

### `[acme.dns]` — DNS provider + credentials

Used only when `challenge` is `dns-01` or `dns-persist-01`.

| Key | Type | Default | Description |
|---|---|---|---|
| `provider` | string | `""` | Any [lego DNS-provider id](https://go-acme.github.io/lego/dns/) (e.g. `cloudflare`, `gandi`, `route53`). |

```toml
[acme.dns]
provider = "gandi"
```

**Credentials are supplied via the environment (or a restricted secrets file), never in the config.**
Each lego provider reads its own variables — for example:

| Provider | Env var(s) |
|---|---|
| `gandi` | `GANDI_API_KEY` |
| `cloudflare` | `CLOUDFLARE_DNS_API_TOKEN` |
| `route53` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` |

See the [lego provider docs](https://go-acme.github.io/lego/dns/) for the exact variable names. A CA
that requires External Account Binding (EAB) takes its EAB key id/HMAC the same way. *(Secret
loading is part of the issuance increment and is not active in the current skeleton.)*

### `[store]` — canonical store

| Key | Type | Default | Description |
|---|---|---|---|
| `path` | string | `/var/lib/syscert` | Where SysCert keeps the source-of-truth cert material and ACME account state. Owned by the `syscert` user; key-bearing files kept `0600`. |

### `[bundle]` — all-in-one file

Controls the composition of `bundle.pem` (see [Output files](#output-files)).

| Key | Type | Default | Description |
|---|---|---|---|
| `order` | list | `["cert","chain","root","key"]` | Components and their order. **Omit a token to exclude it.** Tokens: `cert` (leaf), `chain` (intermediates), `root`, `key`. |

```toml
[bundle]
order = ["key", "cert", "chain"]   # key first, no root
```

> The `root` is automatically dropped when the CA doesn't provide one (public CAs). If `key` is
> present, any target receiving `bundle` must use a non-world-readable mode.

### `[[distribute]]` — delivering to consumers

Zero or more blocks; each copies **one artifact** to a path with the ownership/mode/context that
consumer needs. SysCert overwrites only the paths it manages, and **does not reload consumers** —
have each consumer watch its file (e.g. a `systemd.path` unit) and reload itself.

| Key | Type | Default | Description |
|---|---|---|---|
| `artifact` | string | *(required)* | Which file to place: `cert` \| `privkey` \| `chain` \| `fullchain` \| `bundle`. |
| `path` | string | *(required)* | Destination path. |
| `owner` | string | — | File owner. |
| `group` | string | — | File group. |
| `mode` | string | — | Octal mode, e.g. `"0644"`. **`privkey`/`bundle` hold the key — a world-readable mode is rejected.** |
| `selinux_context` | string | — | Optional SELinux file context (RHEL family), e.g. `cert_t`. |

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

### `[renewal]` / `[logging]`

| Key | Type | Default | Description |
|---|---|---|---|
| `renewal.renew_before` | string | `""` (auto) | Empty = derive the window from the cert's lifetime (short-lived certs renew ~daily; long-lived use a wide window). Set a duration like `"30d"` to override. |
| `logging.level` | string | `info` | Log verbosity. Cert request/renewal events are logged; secret values are always redacted. |

---

## Output files

Per certificate, SysCert writes five PEM files into the store (certbot-compatible names):

| File | Contents | Holds key? |
|---|---|---|
| `cert.pem` | leaf certificate only | no |
| `privkey.pem` | private key | **yes** |
| `chain.pem` | intermediate chain (no leaf, no root) | no |
| `fullchain.pem` | leaf + intermediates (what most servers want) | no |
| `bundle.pem` | configurable all-in-one (default leaf + chain + root + key) | **yes** |

The first four come straight from the ACME response. The **root** in `bundle.pem` is only available
from internal CAs (Vault/step-ca); for public CAs it's omitted.

---

## Full example

A complete config for *internal server → Vault via HTTP-01, delivering to nginx*:

```toml
[cert]
hostname = "app01.internal.lan"
key_type = "ec256"

[acme]
ca            = "vault"
directory_url = "https://vault.example.com:8200/v1/pki/acme/directory"
email         = "ops@example.com"
challenge     = "http-01"

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
mode     = "0600"

[renewal]
renew_before = ""

[logging]
level = "info"
```

Validate it with `syscert dry-run --config ./syscert.toml`.
