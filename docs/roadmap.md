---
title: Roadmap
navLabel: Roadmap
description: What's shipped in syscert today, what's next, and what's planned for the 1.0 line. It's indicative and will change while we're pre-1.0.
order: 10
eyebrow: "// docs · roadmap"
lede: Where syscert is and where it's going. It's pre-1.0, so treat this as the direction of travel, not a promise; it shifts as we learn.
---

> **Status: early (pre-1.0).** Interfaces and defaults can still change between
> minor versions. Have a need or a strong opinion? Open an
> [issue](https://github.com/tfindley/syscert/issues).

## Shipped

- **Full CLI.** The `ensure` default plus `issue`, `renew`, `distribute`, `dry-run`, `void`, `destroy`, and `trust install`/`remove`.
- **CAs.** Let's Encrypt (public), plus HashiCorp Vault PKI and Smallstep step-ca (internal) via `ca = "custom"` + `directory_url`.
- **Challenges.** `dns-01`, `http-01`, and `tls-alpn-01`, with EAB support for the CAs that need it.
- **Delivery.** A canonical store plus per-target distribution that sets the right owner, mode, and SELinux context; certbot-compatible artifacts plus `bundle.pem`.
- **Least privilege.** Runs as a dedicated non-root `syscert` user under a hardened systemd timer, no daemon.
- **Packaging.** `install.sh`, the one-line network installer, and pre-built static Linux binaries (amd64/arm64) with checksums and build provenance.
- **This documentation site**, built from a single canonical Markdown source.

## Next

- **Ansible role.** Fleet installs that run the same steps as `install.sh`, for managing many hosts at once.
- **IP-SAN hardening.** The public-CA `shortlived` profile path and the Vault specifics for certificates with IP Subject Alternative Names (IPv4 is the supported path).
- **Reissue on config drift.** When the certificate's configuration changes (SANs, IP-SANs, key type, or profile), reissue on the next scheduled run instead of waiting for expiry. Right now you apply a changed config by forcing a renewal (`renew --force`); this would let the timer spot the drift and act on its own.

## Planned for 1.0

- **Stabilised config + CLI.** Lock the `syscert.toml` schema and command surface so upgrades stay safe.
- **Broader distro coverage** beyond the tested Debian/Ubuntu and RHEL families.
- **Hardening pass.** Vulnerability and static-analysis gates in CI, an SBOM, and a documented risk review for each release.

## Waiting on upstream

Things we plan to ship but can't yet, because they're blocked on support landing in our dependencies or the CAs. They're listed apart from **Next** because the blocker isn't ours to fix:

- **`dns-persist-01`.** Needs two things. First, the lego ACME library has to expose a non-interactive persistent-DNS provider; current releases ship only a manual/stdin one, which is useless for an unattended service. Second, the CA has to support it, and Let's Encrypt is still rolling it out while HashiCorp Vault's ACME doesn't offer it at all. syscert accepts the config keyword but refuses it with a clear message until both land.

## Not planned

- `device-attest-01`, and acting as a general multi-domain ACME client. syscert is deliberately one cert per host, not a fleet-wide certificate manager.
- A long-running **host service daemon**. For host installs, the model is a systemd timer firing a one-shot binary. (The `--interval` flag gives you an equivalent scheduler for non-systemd contexts like containers and appliances, without changing the host model. See [Containerisation](/docs/containerisation/) and ADR-0046.)

---

See what changed recently in the [changelog](/changelog/), or jump to the [quick start](/docs/quick-start/).
