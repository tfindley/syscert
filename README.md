# SysCert

**Set-and-forget TLS for every machine.** SysCert is a small, least-privilege Linux service that
gets a host its own TLS certificate — from **Let's Encrypt** or an internal **HashiCorp Vault** /
**Smallstep `step-ca`** — keeps it renewed, and delivers it to local consumers (nginx, HAProxy,
Cockpit, databases…) with the exact ownership, mode, and SELinux context each needs. A systemd
timer keeps it fresh forever: no cron, no scripts, no cert babysitting. It's independent of any
host `certbot`.

It speaks ACME via [lego](https://go-acme.github.io/lego/) and writes certbot-compatible output
(`cert.pem` / `privkey.pem` / `chain.pem` / `fullchain.pem`) plus an all-in-one `bundle.pem`.

**Get started in ~5 minutes:** [Install](#install) (build + `install.sh`) → edit two files →
[done](#quick-start). It's just one static binary and a systemd timer.

> **Project status: early (pre-1.0).** Working today: the full `syscert` CLI (the default *ensure*
> plus `issue` / `renew` / `distribute` / `void` / `destroy` / `dry-run` / `trust install`/`remove`),
> the systemd units, and `install.sh`. Not built yet: the Ansible role and published release
> binaries (so for now you build from source — below).

---

## Contents
- [Why / use cases](#why--use-cases)
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

## Why / use cases

You terminate public TLS at the edge (HAProxy, a load balancer). But the hop from there to your
backends — and traffic between internal services — is often plaintext, or relies on hand-made certs
that expire and page someone at 2 a.m. Running an internal CA and then issuing, renewing, copying,
and reloading per-host certs is a chore nobody wants to own. SysCert makes every host responsible
for its own cert, automatically.

- **Encrypt the edge→backend hop** *(the original use case)* — HAProxy handles external TLS; SysCert
  gives the backend its own cert so the HAProxy→backend leg is encrypted too, with no lifecycle to manage.
- **mTLS between services** — run SysCert on both ends against an internal CA: each side has its own
  cert and trusts the other's CA, so services can require + verify client certs. *(The trust-store
  command that completes this is on the roadmap.)*
- **Admin UIs & data stores** — Cockpit, Postgres, Redis, internal APIs, syslog/metrics over TLS.
- **Any host that should just have a valid, auto-renewing cert** without someone owning the renewal.

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

### 3. Install as a service (systemd)

The installer is **external to the binary** — the `syscert` binary never modifies your system; the
script does. It's idempotent (safe to re-run):

```sh
sudo packaging/install.sh ./syscert
```

It creates the `syscert` system user and `/var/lib/syscert` (`0700`), installs the binary to
`/usr/local/bin/syscert`, writes a starter `/etc/syscert/syscert.toml` and a `0640`
`/etc/syscert/secrets` (never overwriting existing files), installs `syscert.service` +
`syscert.timer`, enables the timer, and relabels for SELinux where active.

Then configure and test:

```sh
sudoedit /etc/syscert/syscert.toml      # subject, CA, challenge, distribute targets
sudoedit /etc/syscert/secrets           # e.g. GANDIV5_PERSONAL_ACCESS_TOKEN=...
sudo -u syscert /usr/local/bin/syscert --config /etc/syscert/syscert.toml --staging
systemctl list-timers syscert.timer
```

Uninstall with `sudo packaging/install.sh --uninstall` (add `--purge` to also remove
`/var/lib/syscert`, `/etc/syscert`, and the `syscert` user).

### Manual install (no script)

For unusual systems or if you'd rather not run a script — the steps `install.sh` automates:

```sh
# 1. binary
sudo install -m 0755 syscert /usr/local/bin/syscert
# 2. user + store
sudo groupadd --system syscert
sudo useradd  --system --gid syscert --home-dir /var/lib/syscert \
              --no-create-home --shell /usr/sbin/nologin syscert
sudo install -d -o syscert -g syscert -m 0700 /var/lib/syscert
# 3. config + secrets (edit them; see Configuration below)
sudo install -d /etc/syscert
sudo install -m 0640 -o root -g syscert /dev/null /etc/syscert/secrets
sudo $EDITOR /etc/syscert/syscert.toml
# 4. units (shipped in packaging/systemd/) + enable the timer
sudo cp packaging/systemd/syscert.service packaging/systemd/syscert.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now syscert.timer
# RHEL/SELinux: sudo restorecon -R /var/lib/syscert /etc/syscert
```

> The **Ansible role** for fleet installs is a planned follow-up; it performs these same steps.

### Reference: the user, service, and timer

These are what the installer (or the manual steps) put in place.

**The `syscert` user.** A dedicated, no-login **system user** the service runs as — never root.
It owns the canonical store `/var/lib/syscert` (`0700`) and is granted only `CAP_CHOWN` (via the
unit) so it can set ownership on the copies it distributes. Creating this user — and everything else
that mutates the host — is done by the installer, never by the `syscert` binary.

```sh
groupadd --system syscert
useradd  --system --gid syscert --home-dir /var/lib/syscert --no-create-home \
         --shell /usr/sbin/nologin syscert
```

**`/etc/systemd/system/syscert.service`** — a `oneshot` that runs bare `syscert` (issue/renew as
needed, then distribute), hardened and credential-aware (full file in `packaging/systemd/`):

```ini
[Unit]
Description=SysCert — ensure system TLS certificate is issued, renewed, and distributed
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
User=syscert
Group=syscert
EnvironmentFile=-/etc/syscert/secrets          # DNS/CA creds (0640); optional
ExecStart=/usr/local/bin/syscert --config /etc/syscert/syscert.toml

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/syscert
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
ProtectClock=true
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX

# CAP_CHOWN lets distribution set target ownership.
# Add CAP_NET_BIND_SERVICE only if you use http-01/tls-alpn-01 on :80/:443.
AmbientCapabilities=CAP_CHOWN
CapabilityBoundingSet=CAP_CHOWN
```

**`/etc/systemd/system/syscert.timer`** — what actually *runs* the service: shortly after boot, then
daily with jitter, catching up a missed run. There is **no long-running daemon** — the timer firing
bare `syscert` is the service.

```ini
[Unit]
Description=Run SysCert daily to keep the system certificate fresh

[Timer]
OnBootSec=5min
OnCalendar=daily
RandomizedDelaySec=12h
Persistent=true

[Install]
WantedBy=timers.target
```

Enable it with `sudo systemctl enable --now syscert.timer`; check it with
`systemctl list-timers syscert.timer`.

---

## Quick start

After `install.sh` (above), it has written a starter `/etc/syscert/syscert.toml` and a `0640`
`/etc/syscert/secrets`. Two edits and you're done:

```sh
# 1. set your hostname, CA, and challenge
sudoedit /etc/syscert/syscert.toml

# 2. add provider/CA credentials (kept out of the config). e.g. Gandi LiveDNS:
echo 'GANDIV5_PERSONAL_ACCESS_TOKEN=…' | sudo tee -a /etc/syscert/secrets

# 3. validate offline (no network), then do a real run against LE staging to test
sudo -u syscert syscert dry-run --config-only
sudo -u syscert syscert --staging          # issue + distribute against staging

# happy with it? drop --staging. The systemd timer then keeps it fresh — nothing else to run.
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
| `syscert [--config <path>] [--staging]` | **working** | **The default.** Issue if no cert, renew if due, then distribute. Idempotent — what the systemd timer runs. |
| `syscert issue [--config <path>] [--staging]` | **working** | Obtain a fresh cert + write to the store. **Does not distribute.** |
| `syscert renew [--config <path>] [--staging] [--force]` | **working** | Renew only if due (or `--force`) + write to the store. **Does not distribute.** |
| `syscert distribute [--config <path>]` | **working** | Copy the stored artifacts to the configured targets. |
| `syscert dry-run [--config <path>] [--config-only]` | **working** | Validate config; without `--config-only`, also run the real ACME order/challenge and discard the cert (LE uses staging). |
| `syscert trust install [--ca-file <p>]` / `trust remove` | **working** | Add/remove the internal CA in the **system** trust store (root). Source = `--ca-file` or `acme.ca_bundle`; skips public CAs. |
| `syscert void [--staging] [--force]` | **working** | Revoke the current cert, then reissue + distribute. Interactive unless `--force`. |
| `syscert destroy [--force]` | **working** | Wipe the stored cert + ACME account (provider switch); optionally un-trust an internal CA. Does **not** revoke (use `void`) or reissue. Interactive unless `--force`. |

`--config` defaults to `/etc/syscert/syscert.toml`. `issue`/`renew` update the **store** only — run
`syscert distribute` (or bare `syscert`) to push to consumers.

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
| `ca_bundle` | string | `""` | Path to a PEM of the **internal CA cert(s)** to trust **for the ACME connection only** (not the system store). Use it to bootstrap against a Vault/step-ca whose cert the host doesn't trust yet — SysCert prints a warning when it's set. See below. |

#### Bootstrapping an internal CA the host doesn't trust yet (`ca_bundle`)

Requesting from Vault/step-ca means SysCert makes an HTTPS call to the CA's ACME endpoint, which Go
verifies against the **system** trust store. If the host doesn't trust the internal CA yet, that
call fails with `x509: unknown authority` before anything happens — a chicken-and-egg. Point
`acme.ca_bundle` at the CA's PEM to trust it **for that connection only** (no host changes); SysCert
warns that it's doing so. The full flow:

1. set `acme.ca_bundle = "/etc/syscert/internal-ca.pem"` and issue (`syscert` / `syscert issue`),
2. once the cert is installed, run `sudo syscert trust install` to add the CA root/intermediates to
   the **system** trust store so other local consumers (and clients) trust the issued certs.

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
| `provider` | string | `""` | Any [lego DNS-provider id](https://go-acme.github.io/lego/dns/) (e.g. `cloudflare`, `gandiv5`, `route53`). |

```toml
[acme.dns]
provider = "gandiv5"
```

**Credentials are supplied via the environment (or a restricted secrets file), never in the config.**
Each lego provider reads its own variables — for example:

| Provider | Env var(s) |
|---|---|
| `gandiv5` (Gandi LiveDNS) | `GANDIV5_PERSONAL_ACCESS_TOKEN` |
| `cloudflare` | `CLOUDFLARE_DNS_API_TOKEN` |
| `route53` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` |

> Note: the legacy `gandi` provider uses Gandi's retired XML-RPC API; use **`gandiv5`** (LiveDNS)
> with a **Personal Access Token**.

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
| `logging.level` | string | `info` | `debug` \| `info` \| `warn` \| `error`. |
| `logging.format` | string | `text` | `text` (journald-friendly) \| `json`. |

Operational logs (events + errors, and lego's ACME output) go to **stderr**; command results and
prompts go to **stdout**. Secret values are never logged.

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
