---
title: Roadmap
navLabel: Roadmap
description: What's shipped in syscert today, what's next, and what's planned for the 1.0 line — indicative and subject to change while pre-1.0.
order: 8
eyebrow: "// docs · roadmap"
lede: Where syscert is and where it's going. It's pre-1.0 — this is the direction of travel, not a commitment, and it changes as we learn.
---

> **Status: early (pre-1.0).** Interfaces and defaults can still change between
> minor versions. Have a need or a strong opinion? Open an
> [issue](https://github.com/tfindley/syscert/issues).

## Shipped

- **Full CLI** — the `ensure` default plus `issue`, `renew`, `distribute`,
  `dry-run`, `void`, `destroy`, and `trust install`/`remove`.
- **CAs** — Let's Encrypt (public), HashiCorp Vault PKI, and Smallstep step-ca
  (internal) via `ca = "custom"` + `directory_url`.
- **Challenges** — `dns-01`, `http-01`, `tls-alpn-01`, with EAB support for CAs
  that require it.
- **Delivery** — canonical store + per-target distribution with the right
  owner/mode/SELinux context; certbot-compatible artifacts plus `bundle.pem`.
- **Least privilege** — runs as a dedicated non-root `syscert` user under a
  hardened systemd timer (no daemon).
- **Packaging** — `install.sh`, the one-line network installer, and pre-built
  static Linux binaries (amd64/arm64) with checksums and build provenance.
- **This documentation site**, with a single canonical Markdown source.

## Next

- **Ansible role** — fleet installs that perform the same steps as `install.sh`,
  for managing many hosts at once.
- **IP-SAN hardening** — smoothing the public-CA `shortlived` profile path and the
  Vault IPv4/IPv6 specifics for certificates with IP Subject Alternative Names.
- **`dns-persist-01`** — wired as opt-in and capability-checked; moves from
  experimental to supported as CA availability lands.

## Planned for 1.0

- **Stabilised config + CLI** — lock the `syscert.toml` schema and command surface
  so upgrades are safe.
- **Broader distro coverage** beyond the tested Debian/Ubuntu and RHEL families.
- **Hardening pass** — vulnerability and static-analysis gates in CI, an SBOM, and
  a documented risk review per release.

## Not planned

- `device-attest-01` and acting as a general multi-domain ACME client — syscert is
  deliberately one cert per host, not a fleet-wide certificate manager.
- A long-running daemon — the systemd timer firing a one-shot binary is the model.

---

See what changed recently in the [changelog](/changelog/), or jump to the
[quick start](/docs/quick-start/).
