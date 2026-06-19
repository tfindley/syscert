---
title: Upgrading
navLabel: Upgrading
description: How to upgrade an already-installed syscert to a new version — in place via the one-liner or install.sh, what's preserved vs replaced, how to verify, and how to roll back.
order: 4
eyebrow: "// docs · advanced install · upgrading"
lede: Upgrading an installed syscert is an in-place binary swap — your config, secrets, and certificates are preserved. Here's how, how to verify it, and how to roll back.
---

SysCert upgrades **in place**: you replace the binary (and refresh the systemd units), and your
configuration and certificate state are left untouched. The binary **never self-installs**
(ADR-0034), so there is no `syscert upgrade` command — upgrades are driven by the same installer
you used to set it up.

## The fast path — re-run the one-liner

The network installer always resolves the **latest** release, re-verifies its checksum, and re-runs
`install.sh`:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

Pin a specific version instead of latest:

```sh
SYSCERT_VERSION=v0.3.1 curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

The timer keeps running across the upgrade; the **next scheduled run uses the new binary**. If the
timer was never started, start it once: `sudo systemctl start syscert.timer`.

## What's preserved vs replaced

| Preserved (untouched) | Replaced (refreshed to the new version) |
|---|---|
| `/etc/syscert/syscert.toml` | `/usr/local/bin/syscert` (the binary) |
| `/etc/syscert/secrets` | `/etc/systemd/system/syscert.service` |
| `/etc/default/syscert` | `/etc/systemd/system/syscert.timer` |
| `/var/lib/syscert/` — ACME account key, certificates, archive | (SELinux labels re-applied; `daemon-reload` run) |

Because the store and config survive, the existing **ACME account and certificate carry over** —
the next `syscert ensure` continues issuing/renewing exactly as before.

## From a source checkout or a downloaded binary

If you installed from a built/downloaded binary, point the (idempotent) installer at the new one —
it replaces the binary and units and keeps your config:

```sh
# in a checkout of the new version's packaging
sudo packaging/install.sh ./syscert
```

Fully manual (no `install.sh`): replace the binary, relabel it for SELinux, and refresh the units
only if they changed in the new release:

```sh
sudo install -o root -g root -m 0755 ./syscert /usr/local/bin/syscert
sudo restorecon /usr/local/bin/syscert            # SELinux hosts
sudo systemctl daemon-reload
```

## Verify the upgrade

```sh
syscert version                                   # confirms the new version
sudo -u syscert syscert dry-run --config-only     # your config still validates against the new binary
sudo -u syscert syscert status                    # cert subject, dates, account, targets — read-only
systemctl list-timers syscert.timer               # timer is active and scheduled
```

Verifying the download itself (one-liner does this automatically; manual installs should too):

```sh
sha256sum --check --ignore-missing sha256sums.txt
gh attestation verify syscert-linux-amd64 --repo tfindley/syscert   # SLSA build provenance
```

## Before a notable upgrade

SysCert is pre-1.0, so read the [changelog](/changelog/) first — each release has a **Risk &
Security** note and calls out any behaviour or config changes. If a change touches your config,
rehearse it with `sudo -u syscert syscert dry-run --config-only` (or `--staging` for a full
dry run against Let's Encrypt) before the timer fires.

## Rolling back

Rollback is symmetric — pin the previous tag and re-install:

```sh
SYSCERT_VERSION=v0.3.0 curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

Your config and certificates are preserved either way. The one caveat: if a release migrated the
store or config format (called out in the changelog), check that note before downgrading.

> **Fleet upgrades.** With configuration management, bump the pinned version and re-run — the role
> replaces the binary and units the same way. (The Ansible role tracks the target version as a
> variable.)

---

Next: [Manually](/docs/advanced-install/manually/) · [Configuration](/docs/configuration/) ·
[Changelog](/changelog/)
