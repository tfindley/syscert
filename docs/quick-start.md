---
title: Quick start
navLabel: Quick start
description: Install syscert, edit two files, and have an auto-renewing TLS certificate delivered to your services in about five minutes.
order: 1
eyebrow: "// docs · quick start"
lede: Install the binary, edit two files, and every renewal after that is automatic. About five minutes from nothing to a live, self-renewing certificate.
---

## 1 · Install

One static binary and a systemd timer. The one-liner downloads the release
binary, verifies its checksum, and runs the installer:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

It creates the dedicated `syscert` system user, the canonical store at
`/var/lib/syscert`, a starter config and secrets file, installs the systemd
units, and **enables** (does not start) the timer — so the first run can't fire
against an unconfigured config. Prefer to inspect the script, verify checksums by
hand, or build from source? See [Advanced install](/docs/advanced-install/).

## 2 · Configure two files

The installer writes starters you edit in place. First the config — your subject,
CA, challenge, and where the certificate is delivered:

```toml
# /etc/syscert/syscert.toml
[cert]
hostname = "host.example.com"     # omit to use the system FQDN
sans     = ["api.example.com"]    # optional extra DNS names

[acme]
ca        = "letsencrypt"         # public CA; use "custom" for Vault / step-ca
email     = "you@example.com"
challenge = "dns-01"              # no inbound ports needed

[acme.dns]
provider = "cloudflare"           # any lego DNS provider id

[[distribute]]
artifact = "fullchain"
path     = "/etc/nginx/tls/fullchain.pem"
owner    = "root"
group    = "nginx"
mode     = "0644"

[[distribute]]
artifact = "privkey"
path     = "/etc/nginx/tls/privkey.pem"
owner    = "root"
group    = "nginx"
mode     = "0600"                 # key-bearing → not world-readable
```

Then the credentials your DNS provider needs. These are read from the
environment, never the TOML. Look up the exact variable names for your provider in
the [lego DNS provider list](https://go-acme.github.io/lego/dns/):

```sh
# /etc/syscert/secrets   (env file, 0640 — never put secrets in the .toml)
CLOUDFLARE_DNS_API_TOKEN=your-token-here
```

> **Internal CA?** Set `ca = "custom"` and `directory_url` to your Vault or
> step-ca ACME endpoint — see [Configuration](/docs/configuration/).

## 3 · Validate, then test on staging

Check the config offline first, then do a real run against Let's Encrypt staging
(no rate-limit risk, certs aren't publicly trusted):

```sh
sudo -u syscert syscert dry-run --config-only   # offline checks, no network
sudo -u syscert syscert --staging --env-file /etc/syscert/secrets   # real run; --env-file loads your creds
```

A passing offline check prints the resolved subject, CA, and challenge:

```
config OK:
  subject:   host.example.com
  CA:        letsencrypt
  challenge: dns-01
```

A failing check lists every problem with an actionable message and exits non-zero
— fix those before going live. The full `dry-run` (without `--config-only`)
performs the real ACME order and challenge, then discards the certificate.

## 4 · Go live

Happy with staging? Drop `--staging`, then hand the job to the timer:

```sh
sudo -u syscert syscert              # issue + distribute against production
sudo systemctl start syscert.timer   # hand it over to the timer
systemctl list-timers syscert.timer  # confirm it's scheduled
```

That's it. The timer runs shortly after boot and daily with jitter, renewing when
due and re-delivering to your consumers every time — no cron, no scripts.

---

Next: [Configuration reference](/docs/configuration/) ·
[Distributing certs](/docs/distributing/) ·
[Troubleshooting](/docs/troubleshooting/)
