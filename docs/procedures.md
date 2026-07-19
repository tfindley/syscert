---
title: Procedures
navLabel: Procedures
description: Formal, self-contained operating procedures for syscert, built to drop straight into your own operational documentation.
order: 8.5
eyebrow: "// docs · procedures"
lede: Short, do-this-now SOPs for operating syscert. Each one stands alone and is built to drop into your own ops manual.
---

These are formal operating procedures for syscert, the terse "do this now" layer. They don't replace the rest of the docs. The explanatory *what* and *why* live in [Quick start](/docs/quick-start/), [Configuration](/docs/configuration/), [Advanced install](/docs/advanced-install/), [Distributing certs](/docs/distributing/), and [Troubleshooting](/docs/troubleshooting/), and each procedure links back into those instead of repeating the detail.

Every procedure follows the same fixed structure: purpose, scope, prerequisites, numbered steps with exact commands, verification, and rollback. Copy one straight into a runbook, or reference it from your ops manual.

> **Download all procedures:** [syscert-procedures.zip](/downloads/syscert-procedures.zip) gives you one Markdown file per procedure (named by Procedure ID, frontmatter stripped), ready to drop into your own ops manual.

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
