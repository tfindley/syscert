---
title: Tech stack
navLabel: Tech stack
description: What SysCert is built from — a single static Go binary (no CGO) using the standard library and lego for ACME, run by a systemd timer; plus the documentation site's stack and the build/release pipeline.
order: 2
eyebrow: "// docs · compliance · tech stack"
lede: Everything SysCert is made of — the binary, its dependencies, the runtime model, the docs site, and the build pipeline — in one place.
---

A deliberately small, boring stack: a single static Go binary, a tiny systemd footprint, and a
short dependency list. Nothing to babysit, little to attack.

## The tool

| Area | Choice |
|---|---|
| Language / toolchain | **Go ≥ 1.26**, `CGO_ENABLED=0` → a single **static binary** (linux **amd64** + **arm64**) |
| TLS & crypto | Go standard library `crypto/tls`, `crypto/ecdsa` (P-256 default), `crypto/rand` |
| ACME & DNS | **[lego v5](https://github.com/go-acme/lego)** — the ACME client and DNS-01 providers |
| Config | **[BurntSushi/toml](https://github.com/BurntSushi/toml)** |
| Logging | standard-library structured logging (`log/slog`) to stderr/journal |
| Direct dependencies | **two** (`lego`, `toml`); the large transitive tree is lego's per-provider DNS SDKs |

**Architecture.** A CLI (`syscert ensure|issue|renew|void|destroy|distribute|status|trust|dry-run`)
over small internal packages: config load + **fail-fast validation**, an ACME client wrapper
(lego), an atomic certificate **store** (`/var/lib/syscert`), **distribution** to consumer paths
with per-target owner/mode/SELinux context, **renewal** decisioning, and system **trust**
management. There is **no long-running daemon** — a systemd **oneshot + timer** runs the binary on
a schedule.

**Runtime model.** Runs as a dedicated, non-root **`syscert`** system user under a hardened
systemd unit (`NoNewPrivileges`, `ProtectSystem=strict`, `MemoryDenyWriteExecute`, a single
`CAP_CHOWN`). Outputs are certbot-compatible (`cert`/`privkey`/`chain`/`fullchain`/`bundle`).
See the [Security assessment](/docs/compliance/security/) for the full control set.

## The documentation site

| Area | Choice |
|---|---|
| Generator | **[Astro](https://astro.build)** static site (docs are the canonical `docs/*.md`, vendored at build) |
| Runtime image | **`nginx:alpine-slim`** serving static files (TLS terminated upstream by Traefik) |
| Registry / hosting | image published to **GHCR** (`ghcr.io/tfindley/syscert-web`), pulled by a self-managed host behind **Traefik** |

The website is a *hosting artifact*, not the product — it documents the binary, which is the thing
you install.

## Build, release & CI

| Stage | What runs |
|---|---|
| **CI** (GitHub Actions) | `go build` · `go test` · `go vet` · `gofmt` · `gosec` |
| **Release gating** (`scripts/prerelease.sh`) | the above **plus** `govulncheck`, command/flag↔docs parity, example-config validation, version-stamp checks — all **blocking** |
| **Release** | cross-compiled `amd64`/`arm64` with `-trimpath -ldflags "-s -w"`, `sha256sums.txt`, and a **SLSA build-provenance attestation** (GitHub OIDC) |
| **Site** | container image built and pushed to GHCR on docs/site changes and after each release |

Releases are reproducible (`-trimpath`, pinned `go.sum`, embedded VCS revision) and verifiable
(`sha256sum --check` + `gh attestation verify`).

## Licensing

SysCert is released under the **MIT** license.

---

Next: [Security assessment](/docs/compliance/security/) · [AI-assisted development](/docs/compliance/ai-assisted-development/)
