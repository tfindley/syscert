---
title: Install offline (air-gapped)
navLabel: Offline
description: Install and run SysCert on an air-gapped host with no internet access — build a verified offline bundle on a connected machine, carry it in, verify it, and install with the local packaging installer. No curl | sh and no GitHub reachability required on the target.
order: 1.5
eyebrow: "// docs · advanced install · offline"
lede: Running SysCert offline is easy — the binary only ever talks to your CA. What needs the internet today is the install path. Here's how to remove that and deploy air-gapped.
---

SysCert itself runs fine with no internet. Once installed, it contacts exactly one external thing:
the ACME directory URL you configure. Point that at an **internal CA** — HashiCorp Vault PKI or
step-ca on your own network — and the whole certificate lifecycle happens without ever leaving your
network. There's no telemetry and no update check.

What isn't offline-ready out of the box is *installation*. The one-line installer
(`curl … | sudo sh`) fetches the binary, checksums, and systemd units from GitHub. This page removes
that dependency so you can deploy to a host that has never seen the public internet.

## What needs a network, and what doesn't

| Step | Needs internet? | Offline answer |
|---|---|---|
| Download the binary + units | Yes, normally | Build a bundle on a connected machine (below) |
| Install on the host | **No** — `packaging/install.sh` takes a local binary and needs no network | Run it from the bundle |
| Issue/renew against **Let's Encrypt** | Yes (LE is on the internet) | Use an **internal CA** instead |
| Issue/renew against **Vault / step-ca** | No — it's on your network | Works fully air-gapped |
| Phone home to the maintainer | Never | — |

So "air-gapped SysCert" means: an internal CA, and an install that comes from your own mirror instead
of GitHub.

## Option A — the offline bundle (recommended)

The repo ships a tool that assembles a self-contained, checksum-verified install bundle. Run it once
on any machine that *does* have internet (and the repo checked out):

```sh
scripts/offline-bundle.sh --version v0.4.0 --arch amd64
```

That downloads the release binary and `sha256sums.txt`, verifies the checksum (and the SLSA
provenance, if `gh` is available), pulls the matching systemd packaging pinned to the tag, and writes:

```
syscert-v0.4.0-linux-amd64-offline.tar.gz
```

The tarball contains the binary, `sha256sums.txt`, the `packaging/` directory (installer + units), a
small `install-offline.sh`, and a README. Its own SHA-256 is printed at the end — **write that down**
and carry it out of band, separate from the file.

On the air-gapped host, verify and install:

```sh
# 1. Verify the tarball against the checksum you carried separately
sha256sum syscert-v0.4.0-linux-amd64-offline.tar.gz

# 2. Unpack and install (re-verifies the binary's checksum, then runs packaging/install.sh)
tar xzf syscert-v0.4.0-linux-amd64-offline.tar.gz
cd syscert-v0.4.0-linux-amd64
sudo ./install-offline.sh
```

`install-offline.sh` checks the bundled binary against `sha256sums.txt` before it touches anything,
refuses to install a binary built for the wrong CPU architecture, and then hands off to the standard
[`packaging/install.sh`](/docs/advanced-install/manually/) — which creates the `syscert` user, lays
down `/var/lib/syscert` and `/etc/syscert`, installs the units, and enables (does not start) the
timer.

Then configure it for your environment (next section) before the timer's first run.

## Option B — fully manual (no bundle tool)

If you'd rather not run the tool, do the same thing by hand. On a connected machine, from the
[release](https://github.com/tfindley/syscert/releases) you want:

```sh
ver=v0.4.0 arch=amd64
base=https://github.com/tfindley/syscert/releases/download/$ver

# Binary + checksums
curl -fsSLO $base/syscert-linux-$arch
curl -fsSLO $base/sha256sums.txt
grep " syscert-linux-$arch\$" sha256sums.txt | sha256sum -c -   # must print: OK
gh attestation verify syscert-linux-$arch --repo tfindley/syscert   # optional, recommended

# Packaging pinned to the same tag
raw=https://raw.githubusercontent.com/tfindley/syscert/$ver/packaging
mkdir -p packaging/systemd
curl -fsSL $raw/install.sh              -o packaging/install.sh
curl -fsSL $raw/systemd/syscert.service -o packaging/systemd/syscert.service
curl -fsSL $raw/systemd/syscert.timer   -o packaging/systemd/syscert.timer
```

Carry all of it to the target host (keeping `packaging/install.sh` next to its `systemd/` directory),
re-verify the checksum there, then:

```sh
sudo packaging/install.sh ./syscert-linux-amd64
```

You can also build the binary yourself instead of downloading it — see
[Compile from source](/docs/advanced-install/compile-from-source/) — which removes any trust in the
published artefact. The install step is identical; just pass the path to the binary you built.

## Configuring for an offline / internal CA

The install lays down a starter config; edit `/etc/syscert/syscert.toml` for your network:

- **Point at your internal CA.** Set `ca = "custom"` and
  `directory_url = "https://vault.internal:8200/v1/pki/acme/directory"` (or your step-ca directory).
  See the [Configuration reference](/docs/configuration/) and the
  [Vault examples](/docs/examples/vault-dns-01/).
- **Trust the internal CA's root.** Your CA almost certainly isn't in the public trust store. Use
  `acme.ca_bundle` for connection-only trust to bootstrap, or install the root system-wide with
  `sudo syscert trust install` (see [Trust an internal CA](/docs/procedures/trust-internal-ca/)).
  Your certificate *consumers* need that root too.
- **Use a challenge that works on your network.** `dns-01` against internal DNS is the default;
  `http-01` / `tls-alpn-01` work if the CA can reach the host. IP-SANs force the HTTP/ALPN
  challenges automatically.
- **Provide secrets the offline way.** DNS/CA credentials come from the environment or a `0640`
  secrets file, never the TOML — exactly as on a connected host.

Then test once against your CA and enable the timer, per the
[Quick start](/docs/quick-start/) and [Install & deploy procedure](/docs/procedures/install-and-deploy/).

## Offline upgrades

Upgrading is the same shape as installing: build a fresh bundle for the new tag on a connected
machine, carry it in, and re-run it. `install.sh` replaces the binary and units in place and leaves
`/var/lib/syscert` and `/etc/syscert` untouched, so certificates and config survive. See
[Upgrading](/docs/advanced-install/upgrading/) for the full flow and rollback notes.

## A note on Ansible

The [Ansible role](/docs/roadmap/) will grow a first-class offline mode later (it's being built on a
separate branch). Until then, the offline bundle plus `packaging/install.sh` is the supported
air-gapped path — and it's the same installer the role will wrap, so nothing you set up here is
throwaway.
