---
title: Comparison
navLabel: Comparison
description: How SysCert compares to certbot and the lego CLI — and, briefly, Caddy/Traefik, cert-manager, and acme.sh — for the job it's built for, a single system-level certificate delivered to a host's own services, with an honest "when to use which".
order: 1.5
eyebrow: "// docs · comparison"
lede: SysCert isn't trying to replace certbot. Here's the specific job it's built for — one host's system certificate, usually from internal PKI, delivered to local services with the right ownership and SELinux — and an honest guide to when another tool fits better.
---

SysCert hands the ACME protocol to [lego](https://github.com/go-acme/lego), the same way certbot hands it to its own ACME code — and because lego also ships a standalone CLI, that pairing gets its own comparison below. So the *plumbing* isn't where these tools differ. What differs is the **operational model** wrapped around it. This page compares them for the job SysCert is built for; if your job is different, the guidance at the bottom points you elsewhere.

## The job: one host's system certificate

A single machine needs **one OS-level TLS certificate** for that machine's **own system services**. It's often issued by an **internal CA** like HashiCorp Vault or step-ca, and it has to land at each service's expected path with the right **owner, mode, and SELinux context**. Not a fleet-wide certificate manager, not a public CDN edge. Just the host's own identity certificate.

## SysCert vs certbot

| For this job… | certbot | SysCert |
|---|---|---|
| **Get the cert** | ✓ `certonly` (standalone / dns / webroot) | ✓ dns-01 default (http-01 / tls-alpn-01 too) |
| **From internal PKI (Vault / step-ca)** | Possible via `--server`, but Let's-Encrypt-shaped ergonomics | First-class: `ca = "custom"` + `directory_url`, `ca_bundle` bootstrap, `trust install`/`remove` |
| **Deliver it to the services that consume it** | You write it — `--deploy-hook` copy scripts, per service | Declarative `[[distribute]]` — each target's path, **owner, mode, SELinux context** |
| **Perms / ownership / SELinux on delivered files** | Manual, in your hooks | Enforced from config (key-bearing files `0600`) |
| **Runs as** | **root** | dedicated **non-root** `syscert` user (single `CAP_CHOWN`) |
| **Getting services to pick up the new cert** | root `--deploy-hook` / `--renew-hook` **scripts** (executes commands) | **none** — each consumer watches its file and reloads itself (SysCert runs no commands) |
| **Output layout** | Let's Encrypt `live/` layout | certbot-compatible **plus** a configurable `bundle.pem` (leaf/chain/key, in the order a service wants) |
| **Config model** | flags + per-cert renewal conf | one **TOML** describing the whole lifecycle |
| **Footprint** | Python interpreter + dependency tree (or a snap) | one static binary, nothing to install |
| **Pre-flight** | fails at issuance time | `dry-run --config-only` — offline, fail-fast |
| **Supply chain** | distro / pip package | SLSA build provenance, SBOM, gosec + govulncheck gates, a published [security assessment](/docs/compliance/security/) |
| **Rolling it out to many hosts** | no first-party config-management story — you wrap it yourself | in-tree **Ansible role**, running the same steps as `install.sh` across an inventory |
| **Monitoring** | exit codes and logs; the rest is yours to build | optional Prometheus node_exporter textfile + Ansible facts (`[observe]`), off by default |
| **Ubiquity / maturity** | **the standard**, decade-proven | bespoke, pre-1.0 |

certbot *gets you the cert*. For a single system certificate, you then own the glue: put it where each daemon reads it, with the right perms and SELinux, from root, then re-run a reload script. SysCert makes that part **declarative and least-privilege**, and it treats the **internal-CA source as the default** rather than an afterthought. The honest framing isn't that certbot can't do it. With certbot, this host accretes a pile of root deploy-hooks and copy scripts; with SysCert, it's one config file, running unprivileged, that never executes a command.

Where certbot is the better call, and this is the part that keeps the rest honest: if that single certificate is a **public Let's Encrypt cert for one web server** reachable on :80, certbot's `standalone` / `--nginx` / `--apache` plugins are simpler and more conventional. certbot is also the tool everyone already knows, packaged everywhere, proven for a decade. Choosing SysCert means *you* own and maintain a certificate tool, and that's a real cost certbot doesn't carry.

## SysCert vs lego CLI

First, the disclosure that shapes everything below: SysCert doesn't compete with lego — it's **built on** it. The lego *library* is SysCert's embedded ACME engine, which is why the two agree on every protocol-level fact: same challenges, the same 218 DNS providers, the same code talking to the CA. But the lego project also ships its own **CLI**, a standalone client in the same "get the cert, you handle delivery" family as certbot, and *that's* what this table compares. The plumbing is identical by construction; every row is about what's wrapped around it.

| For this job… | lego CLI | SysCert |
|---|---|---|
| **Get the cert** | ✓ `lego run` — dns-01 / http-01 / tls-alpn-01, 218 DNS providers | ✓ the **same engine**, same challenges, same 218 providers |
| **From internal PKI (Vault / step-ca)** | Works — `--server` plus the `LEGO_CA_CERTIFICATES` env var — but Let's-Encrypt-shaped ergonomics | First-class: `ca = "custom"` + `directory_url`, connection-only `ca_bundle`, `trust install`/`remove` |
| **Deliver it to the services that consume it** | You write it — `--pre-hook` / `--deploy-hook` / `--post-hook` scripts | Declarative `[[distribute]]` — each target's path, **owner, mode, SELinux context**, written atomically |
| **Perms / ownership / SELinux on delivered files** | `.lego/` storage is `0700`; beyond that, whatever your hooks do | Enforced from config (key-bearing files `0600`) |
| **Runs as** | whichever user invokes it — in practice **root**, once hooks need to chown | dedicated **non-root** `syscert` user (single `CAP_CHOWN`) |
| **Getting services to pick up the new cert** | hook **scripts** (executes commands) | **none** — each consumer watches its file and reloads itself (SysCert runs no commands) |
| **Renewal scheduling** | bring your own cron or timer; renewal timing is **ARI**-aware | shipped hardened **systemd timer** (or `--interval` for containers) — though renewal is a plain time window, **no ARI yet** |
| **Output layout** | `.lego/certificates/` — `.crt` / `.key` / `.issuer.crt`, optional PFX | certbot-shaped five artifacts **plus** a configurable `bundle.pem` |
| **Client breadth** | **wider**: multiple certs per host, CSR input, PFX, `--preferred-chain`, `--must-staple`, account key rollover | deliberately **one cert per host** — none of those, by design |
| **Pre-flight / inspection** | `lego certificates list`; config problems surface at run time | offline `dry-run --config-only` and `status` — fail-fast, no network |
| **Rolling it out to many hosts** | no first-party config-management story | in-tree **Ansible role**, variables mirroring the TOML one-for-one |
| **Monitoring** | exit codes and logs | optional Prometheus textfile + Ansible facts (`[observe]`), off by default |
| **Footprint** | one static Go binary | one static Go binary — genuinely **no difference** |

The pattern is the certbot comparison again, only closer to home: the lego CLI *gets you the cert* — with the identical engine SysCert uses — and delivery is hook scripts you write and run yourself, usually as root. SysCert trades the CLI's breadth for the delivery, privilege, and lifecycle opinions this page is about. Where the engine is shared, the honest claim isn't "better ACME"; it's that the glue around the ACME is the product.

Where the lego CLI is the better call: it's simply the **more general client**. Several certificates on one host, CSR input, PFX for Windows-shaped consumers, `--preferred-chain`, `--must-staple`, account key rollover, ARI-aware renewal timing — SysCert has none of those, deliberately, and one of them (ARI) it should honestly grow. If you're comfortable owning the deploy hooks and the timer, the lego CLI is the leaner tool, maintained by the same people who maintain the engine — choosing SysCert over it means preferring our wrapper's opinions to your own scripts.

## Other tools, briefly

**Caddy / Traefik.** If a web server or reverse proxy *is* the consumer and it's web-fronted, they do ACME **natively**: they issue, renew, and reload themselves, no external agent. Great at that. But they aren't a general "deliver a cert to arbitrary system services" tool, and their internal-CA story is thinner.

**cert-manager** is the **Kubernetes** answer: `Certificate` CRDs, issuers including Vault. If your certificates live in a cluster, use it. SysCert is for **hosts, VMs, appliances, and standalone containers** outside an orchestrator.

**acme.sh** is a capable pure-shell ACME client with strong dns-01 coverage, lighter than certbot. But it's the same "get the cert, you handle delivery" shape, as a shell script rather than a single static binary with an opinionated distribution and trust model.

## When to use which

**Reach for certbot / Caddy / Traefik** when: it's a **public Let's Encrypt** cert for a **single web server** reachable on :80/:443, http-01/webroot is your validation path, and you want the standard, ubiquitous tool.

**Reach for cert-manager** when: you're in **Kubernetes**.

**Reach for the lego CLI** when: you need **more than one certificate** on the host, or client features SysCert deliberately omits — CSR input, PFX, `--preferred-chain`, key rollover — and you're happy writing the delivery hooks and scheduling renewals yourself.

**SysCert fits** when several of these are true:

- the certificate comes from **internal PKI** (Vault / step-ca);
- it's an **OS-level host certificate** consumed by one or more **local services**;
- it must be **delivered with per-target owner / mode / SELinux**;
- you want a **single static binary** that runs **unprivileged** and **never executes commands**;
- you're doing this on **more than one host** and want the rollout itself managed (the Ansible role), not just the certificate;
- **supply-chain / compliance posture** (attested builds, least privilege, no command-execution surface) matters.

Where certbot's envelope fits, use certbot. Where the job is a host's own certificate from internal PKI, delivered to its services with strict permissions and a clean audit story, that's what SysCert is opinionated about.

---

Next: [Quick start](/docs/quick-start/) · [Configuration](/docs/configuration/) · [Distributing certificates](/docs/distributing/) · [Security assessment](/docs/compliance/security/)
