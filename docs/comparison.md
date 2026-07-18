---
title: Comparison
navLabel: Comparison
description: How SysCert compares to certbot — and, briefly, Caddy/Traefik, cert-manager, and acme.sh — for the job it's built for, a single system-level certificate delivered to a host's own services, with an honest "when to use which".
order: 1.5
eyebrow: "// docs · comparison"
lede: SysCert isn't trying to replace certbot. Here's the specific job it's built for — one host's system certificate, usually from internal PKI, delivered to local services with the right ownership and SELinux — and an honest guide to when another tool fits better.
---

SysCert delegates the ACME protocol to [lego](https://github.com/go-acme/lego), the same way certbot
delegates it to its own ACME implementation — so the *plumbing* isn't the differentiator. What differs
is the **operational model** wrapped around it. This page compares the tools for the job SysCert is
built for; if your job is different, the guidance at the bottom points you elsewhere.

## The job: one host's system certificate

A single machine needs **one OS-level TLS certificate**, consumed by that machine's **own system
services** — often issued by an **internal CA** (HashiCorp Vault, step-ca), and delivered to each
service's expected path with the right **owner, mode, and SELinux context**. Not a fleet-wide
certificate manager, not a public CDN edge — the host's own identity certificate.

## SysCert vs certbot

| For this job… | certbot | SysCert |
|---|---|---|
| **Get the cert** | ✓ `certonly` (standalone / dns / webroot) | ✓ dns-01 default (http-01 / tls-alpn-01 too) |
| **From internal PKI (Vault / step-ca)** | Possible via `--server`, but Let's-Encrypt-shaped ergonomics | First-class: `ca = "custom"` + `directory_url`, `ca_bundle` bootstrap, `trust install`/`remove` |
| **Deliver it to the services that consume it** | You write it — `--deploy-hook` copy scripts, per service | Declarative `[[distribute]]` — each target's path, **owner, mode, SELinux context** |
| **Perms / ownership / SELinux on delivered files** | Manual, in your hooks | Enforced from config (key-bearing files `0600`) |
| **Runs as** | **root** | dedicated **non-root** `syscert` user (single `CAP_CHOWN`) |
| **Getting services to pick up the new cert** | root `--deploy-hook` / `--renew-hook` **scripts** (executes commands) | **none** — each consumer watches its file and reloads itself (SysCert runs no commands) |
| **Output layout** | Let's Encrypt `live/` layout | certbot-compatible **plus** a configurable `bundle.pem` (leaf/chain/root/key, in the order a service wants) |
| **Config model** | flags + per-cert renewal conf | one **TOML** describing the whole lifecycle |
| **Footprint** | Python interpreter + dependency tree (or a snap) | one static binary, nothing to install |
| **Pre-flight** | fails at issuance time | `dry-run --config-only` — offline, fail-fast |
| **Supply chain** | distro / pip package | SLSA build provenance, SBOM, gosec + govulncheck gates, a published [security assessment](/docs/compliance/security/) |
| **Ubiquity / maturity** | **the standard**, decade-proven | bespoke, pre-1.0 |

**The essence.** certbot *gets you the cert*. For a single system certificate you then own the
"put it where each daemon reads it, with the right perms and SELinux, from root, and re-run a reload
script" part as bespoke glue. SysCert makes exactly that part **declarative and least-privilege**, and
treats the **internal-CA source as the default**, not an afterthought. So the honest framing isn't
"certbot can't" — it's *"with certbot this host accretes a pile of root deploy-hooks and copy scripts;
with SysCert it's one config file, running unprivileged, that never executes a command."*

**Where certbot is the better call — and this is the part that keeps the rest credible:** if that
single certificate is a **public Let's Encrypt cert for one web server** reachable on :80, certbot's
`standalone` / `--nginx` / `--apache` plugins are simpler and more conventional — and certbot is the
tool everyone already knows, packaged everywhere, proven for a decade. Choosing SysCert means *you*
own and maintain a certificate tool; that's a real cost certbot doesn't carry.

## Other tools, briefly

- **Caddy / Traefik** — if a web server or reverse proxy *is* the consumer and it's web-fronted, they
  do ACME **natively** (issue, renew, and reload themselves, no external agent). Excellent for that;
  they aren't a general "deliver a cert to arbitrary system services" tool, and their internal-CA
  story is thinner.
- **cert-manager** — the **Kubernetes** answer (`Certificate` CRDs, issuers including Vault). If your
  certificates live in a cluster, use it. SysCert is for **hosts, VMs, appliances, and standalone
  containers** — outside an orchestrator.
- **acme.sh** — a capable pure-shell ACME client with strong dns-01 coverage; lighter than certbot,
  but the same "get the cert, you handle delivery" shape, as a shell script rather than a single
  static binary with an opinionated distribution + trust model.

## When to use which

**Reach for certbot / Caddy / Traefik** when: it's a **public Let's Encrypt** cert for a **single web
server** reachable on :80/:443, http-01/webroot is your validation path, and you want the
standard, ubiquitous tool.

**Reach for cert-manager** when: you're in **Kubernetes**.

**SysCert fits** when several of these are true:

- the certificate comes from **internal PKI** (Vault / step-ca);
- it's an **OS-level host certificate** consumed by one or more **local services**;
- it must be **delivered with per-target owner / mode / SELinux**;
- you want a **single static binary** that runs **unprivileged** and **never executes commands**;
- **supply-chain / compliance posture** (attested builds, least privilege, no command-execution
  surface) matters.

Where certbot's envelope fits, use certbot. Where the job is a host's own certificate from internal
PKI, delivered to its services with strict permissions and a clean audit story — that's the job
SysCert is opinionated about.

---

Next: [Quick start](/docs/quick-start/) · [Configuration](/docs/configuration/) ·
[Distributing certificates](/docs/distributing/) · [Security assessment](/docs/compliance/security/)
