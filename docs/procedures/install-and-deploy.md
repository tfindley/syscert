---
title: "SC-OPS-001: Install & deploy"
navLabel: "001 · Install & deploy"
description: Formal procedure for installing syscert on a new host — one-line network installer or manual verified-binary path — and starting the systemd timer.
order: 1
eyebrow: "// docs · procedures · SC-OPS-001"
lede: Get syscert installed, configured, and running on a new host. Two supported methods — the one-line network installer and the manual verified-binary path.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-001 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` (install steps) and the `syscert` service user (validation steps) |
| **Last reviewed** | 2026-06-22 |

## Purpose

Install syscert on a host for the first time, configure it to issue and distribute a certificate,
validate the configuration, and hand operation over to the systemd timer.

## Scope

Covers Debian/Ubuntu and the RHEL family (amd64/arm64). Two supported install methods:

- **(A)** the one-line network installer — the normal path.
- **(B)** manual verified-binary install — for air-gapped or inspect-first environments.

**Not covered:** compile-from-source (see [Compile from source](/docs/advanced-install/compile-from-source/)),
cron-only installs (see [As a cron job](/docs/advanced-install/cron/)), and Ansible fleet installs
(planned — see [roadmap](/docs/roadmap/)).

## Prerequisites

- Root access (or `sudo`) on the target host.
- The host has a resolvable FQDN (`hostname -f` returns a full name), or you intend to set
  `hostname` explicitly in the config.
- Outbound HTTPS access (port 443) to your chosen CA's ACME endpoint.
- DNS provider credentials ready (for `dns-01`), or inbound port 80/443 open (for `http-01` /
  `tls-alpn-01`).
- For **method B**: `curl`, `sha256sum`, and `git` available on the host.

## Procedure

### Method A — one-line network installer

**1. Run the installer.**

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

The installer creates the `syscert` system user, `/var/lib/syscert` (mode `0700`), a starter
`/etc/syscert/syscert.toml` (`0640 root:syscert`), a `0640` `/etc/syscert/secrets`, the
`/etc/default/syscert` operator settings file, installs the systemd units, enables (but does not
start) the timer, and applies SELinux labels where active.

**2. Edit the configuration.**

```sh
sudo vi /etc/syscert/syscert.toml
```

Set at minimum: `[cert] hostname`, `[acme] ca`, `[acme] email`, `[acme] challenge`, and at least
one `[[distribute]]` block. See [Configuration](/docs/configuration/) for the full reference and
[examples/](https://github.com/tfindley/syscert/tree/main/examples) for ready-to-edit starters.

**3. Add credentials to the secrets file.**

```sh
sudo vi /etc/syscert/secrets
```

Add the environment variables your DNS provider needs (e.g. `CLOUDFLARE_DNS_API_TOKEN=…`). The
exact variable names are listed in the [lego DNS provider docs](https://go-acme.github.io/lego/dns/).
Secrets never go in the TOML — see [Configuration](/docs/configuration/).

**4. Validate the config offline.**

```sh
sudo -u syscert syscert dry-run --config-only
```

Expected output:

```
config OK:
  subject:   host.example.com
  CA:        letsencrypt
  challenge: dns-01
```

Fix any reported errors before continuing.

**5. Test against the CA's staging environment.**

```sh
sudo -u syscert syscert --staging --env-file /etc/syscert/secrets
```

This performs a real ACME order using the staging CA (no rate-limit risk, certificate is not
publicly trusted). Confirm the certificate is issued and distributed to the configured targets.

**6. Start the timer.**

```sh
sudo systemctl start syscert.timer
```

Skip to **Verification** below.

---

### Method B — manual verified-binary install

**1. Download the release binary and verify it.**

```sh
# amd64 — for arm64 use syscert-linux-arm64
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/syscert-linux-amd64 -o syscert
chmod +x syscert

# Verify against the published checksums
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/sha256sums.txt -o sha256sums.txt
sha256sum --check --ignore-missing sha256sums.txt

./syscert --help
```

To pin a specific version, replace `latest/download` with `download/<tag>` (e.g. `download/v0.3.0`).

**2. Clone the packaging files and run the installer.**

```sh
# clone into a named dir so it doesn't collide with the ./syscert binary
git clone https://github.com/tfindley/syscert.git syscert-src
sudo syscert-src/packaging/install.sh ./syscert
```

The installer is idempotent and external to the binary — it creates the system user, store, config
starters, systemd units, and SELinux labels. The binary never self-installs.

**3–6. Follow steps 2–6 from Method A** (edit config, add credentials, validate, test staging, start
the timer).

## Verification

```sh
systemctl list-timers syscert.timer            # timer is active and scheduled
syscert version                                # prints the installed version
sudo -u syscert syscert dry-run --config-only  # config validates cleanly
sudo -u syscert syscert status                 # shows cert subject, expiry, account, targets
```

Confirm the distributed artifacts exist at the paths in your `[[distribute]]` blocks and have the
expected owner/mode:

```sh
ls -l /etc/nginx/tls/fullchain.pem   # adjust to your configured path(s)
```

## Rollback / recovery

The installer is idempotent — re-running it is safe. To fully remove the installation:

```sh
# Keep data (config, certificates)
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall

# Remove everything including /var/lib/syscert, /etc/syscert, and the syscert user
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge
```

`--purge` asks for confirmation on the terminal; set `SYSCERT_ASSUME_YES=1` to skip.

## Related procedures

- [SC-OPS-002 — Change certificate details & reissue](/docs/procedures/change-cert-details/) — once
  installed, if you need to add SANs or change the key type.
- [SC-OPS-009 — Upgrade syscert](/docs/procedures/upgrade/) — in-place binary swap.
- [SC-OPS-010 — Uninstall or purge](/docs/procedures/uninstall/) — full removal.

**Explanatory docs:** [Quick start](/docs/quick-start/) · [Advanced install](/docs/advanced-install/) ·
[Advanced install → Manually](/docs/advanced-install/manually/) · [Configuration](/docs/configuration/)
