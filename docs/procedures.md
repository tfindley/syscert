---
title: Procedures
navLabel: Procedures
description: Formal, self-contained operating procedures for syscert — designed to be importable into your own operational documentation.
order: 7
eyebrow: "// docs · procedures"
lede: Concise, action-oriented SOPs for operating syscert. Each procedure is self-contained and designed to be imported into your own ops manual.
---

These are formal operating procedures for syscert — the terse "do this now" layer. They are not a
substitute for the rest of the documentation: the explanatory *what* and *why* live in
[Quick start](/docs/quick-start/), [Configuration](/docs/configuration/),
[Advanced install](/docs/advanced-install/), [Distributing certs](/docs/distributing/), and
[Troubleshooting](/docs/troubleshooting/). Each procedure cross-links into those docs rather than
repeating the detail.

Each procedure is self-contained with a fixed structure: purpose, scope, prerequisites, numbered
steps with exact commands, verification, and rollback. They are intended to be copied or referenced
directly from an operational runbook or ops manual.

> **Combined download:** a single-file Markdown bundle of all procedures will be available here.

## Procedure index

| ID | Procedure | When you'd use it |
|---|---|---|
| [SC-OPS-001](/docs/procedures/install-and-deploy/) | Install & deploy | Setting up syscert on a new host for the first time. |
| [SC-OPS-002](/docs/procedures/change-cert-details/) | Change certificate details & reissue | Adding or removing SANs, changing the key type or issuance profile, then forcing a fresh certificate. |
| [SC-OPS-003](/docs/procedures/force-renewal/) | Force an immediate renewal | Renewing the certificate right now without any config change — expiry bypass. |
| [SC-OPS-004](/docs/procedures/rotate-key/) | Rotate the private key | Rotating to a fresh keypair, or clearing `reuse_key` to do so explicitly. |
| [SC-OPS-005](/docs/procedures/revoke-and-replace/) | Revoke and replace (`void`) | Revoking the current certificate at the CA and immediately reissuing a replacement. |
| [SC-OPS-006](/docs/procedures/migrate-ca/) | Migrate to a different CA | Switching from one ACME CA to another — new account, fresh order, optional EAB. |
| [SC-OPS-007](/docs/procedures/manage-distribution-targets/) | Manage distribution targets | Adding, changing, or removing `[[distribute]]` blocks and pushing the new delivery config. |
| [SC-OPS-008](/docs/procedures/trust-internal-ca/) | Trust an internal CA system-wide | Installing or removing an internal CA's root in the system trust store. |
| [SC-OPS-009](/docs/procedures/upgrade/) | Upgrade syscert | In-place binary swap to a new version via the one-liner or install.sh. |
| [SC-OPS-010](/docs/procedures/uninstall/) | Uninstall or purge | Removing syscert — keeping data with `--uninstall` or wiping everything with `--purge`. |
| [SC-OPS-011](/docs/procedures/recover/) | Recover from a broken state | Wiping all cert state with `destroy` and re-provisioning from scratch. |
