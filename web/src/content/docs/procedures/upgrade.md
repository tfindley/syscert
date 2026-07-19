---
title: "SC-OPS-009: Upgrade syscert"
navLabel: "009 · Upgrade syscert"
description: How to upgrade an already-installed syscert to a new version, an in-place binary swap via the one-liner or install.sh, with verification and rollback.
order: 9
eyebrow: "// docs · procedures · SC-OPS-009"
lede: Upgrading is an in-place binary swap. Config, secrets, and certificates are preserved. The timer keeps running and uses the new binary on its next fire.
---

| | |
|---|---|
| **Procedure ID** | SC-OPS-009 |
| **Applies to** | syscert ≥ v0.3 |
| **Audience** | `root` |
| **Last reviewed** | 2026-06-22 |

## Purpose

Swap the installed syscert binary for a new version (and refresh the systemd units), while leaving your configuration, secrets, ACME account, and certificates untouched.

## Scope

Covers the two upgrade paths: the one-line network installer, which is the normal one, and the manual binary swap for air-gapped or inspect-first setups. The binary never self-installs; the installer always drives an upgrade.

For the full picture of what's kept versus replaced, see [Advanced install → Upgrading](/docs/advanced-install/upgrading/).

## Prerequisites

- Root access on the host.
- For the manual path: the new binary downloaded and verified (see step 1B).
- Read the [changelog](/changelog/) for the version you're moving to. Every release carries a **Risk & Security** note and flags any config changes you need to handle before the timer fires.

## Procedure

### Method A — one-line installer (recommended)

**1A. Re-run the network installer.**

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

The installer pulls the latest binary, checks its checksum, replaces `/usr/local/bin/syscert`, refreshes the systemd units, and re-applies SELinux labels. Config, secrets, and the store stay untouched.

To pin a specific version instead of latest:

```sh
SYSCERT_VERSION=v0.3.1 curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

Skip to **Verification** below.

---

### Method B — manual binary swap

**1B. Download and verify the new binary.**

```sh
# amd64 — for arm64 use syscert-linux-arm64
curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/syscert-linux-amd64 -o syscert
chmod +x syscert

curl -fsSL https://github.com/tfindley/syscert/releases/latest/download/sha256sums.txt -o sha256sums.txt
sha256sum --check --ignore-missing sha256sums.txt
```

**2B. Run the installer against the new binary.**

```sh
git clone https://github.com/tfindley/syscert.git syscert-src
sudo syscert-src/packaging/install.sh ./syscert
```

Or go fully manual (no `install.sh`, and only if the units haven't changed):

```sh
sudo install -o root -g root -m 0755 ./syscert /usr/local/bin/syscert
sudo restorecon /usr/local/bin/syscert    # SELinux hosts only
sudo systemctl daemon-reload
```

## Verification

```sh
syscert version                                   # confirms the new version string
sudo -u syscert syscert dry-run --config-only     # config still validates against the new binary
sudo -u syscert syscert status                    # cert subject, dates, account, targets — read-only
systemctl list-timers syscert.timer               # timer is active and scheduled
```

## Rollback / recovery

Pin the previous version and re-run the installer:

```sh
SYSCERT_VERSION=v0.3.0 curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

Either way, config and certificates are preserved. If the release notes mentioned a store or config format change, read them before you downgrade.

## Related procedures

- [SC-OPS-001 — Install & deploy](/docs/procedures/install-and-deploy/) — initial installation reference.
- [SC-OPS-010 — Uninstall or purge](/docs/procedures/uninstall/) — full removal.

**Explanatory docs:** [Advanced install → Upgrading](/docs/advanced-install/upgrading/) · [Changelog](/changelog/)
