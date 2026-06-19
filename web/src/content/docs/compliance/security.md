---
title: Security assessment
navLabel: Security
description: SysCert's published security assessment and risk register — static analysis (go vet, gosec, govulncheck), binary hardening, controls by domain, findings, residual-risk register, and control-framework mapping.
order: 1
eyebrow: "// docs · compliance · security"
lede: A published, tool-backed security assessment of the SysCert Go application and release binary — with a full risk register, so you can review the posture before you deploy.
---

This assessment is published for transparency. It covers the **Go application and the released
binary**; it is re-run as part of every release (`prerelease.sh` gates `gosec`, `govulncheck`, and
the test suite). To report a vulnerability, see the [Security policy](https://github.com/tfindley/syscert/blob/main/SECURITY.md).

| | |
|---|---|
| **Version assessed** | v0.3.1 (git `8a30a15`, `vcs.modified=false`) |
| **Assessment date** | 2026-06-19 |
| **Method** | Tool-assisted static analysis + manual review (see §2) |
| **Distribution** | Public |
| **Overall posture** | **Low risk.** Memory-safe Go, no CGO, least-privilege runtime, secrets never logged, fail-fast validation, signed-provenance releases. Zero findings from `go vet`, `gosec`, and `govulncheck`. |

> Scope note: the Ansible role lives in a separate repository and is **out of scope** here.
> `install.sh`/`net-install.sh` (shell) and the systemd units are covered only where they define
> the binary's runtime privilege/permissions context.

## 1. Executive summary

SysCert is a small, single-purpose Go program that obtains an ACME/TLS certificate for a host
(Let's Encrypt, HashiCorp Vault PKI, or Smallstep step-ca), stores it under a locked
syscert-owned directory, and copies the requested artefacts to local consumers with explicit
ownership/mode/SELinux context. It runs **as a dedicated non-root system user on a systemd
timer** — there is no long-running daemon and no inbound network surface in the default
(dns-01) configuration.

The security posture is **strong for its size and stage**:

- **Automated analysis is clean.** `go vet` (clean), `gosec` (**0 issues**, 9 reviewed
  `#nosec` suppressions), and `govulncheck` (**0 vulnerabilities** across 676 modules).
- **Memory-safe, statically linked.** Pure Go, `CGO_ENABLED=0` → no C/libc memory-safety class
  of defects and no shared-library hijack surface.
- **Secrets are handled correctly.** DNS/CA credentials and the EAB HMAC come from the
  environment / a `0640` file, are **never written to the config, logged, or printed**, and this
  is enforced by a regression test.
- **Least privilege at runtime.** Dedicated `syscert` user, a single ambient capability
  (`CAP_CHOWN`), and an extensively hardened systemd unit (`ProtectSystem=strict`,
  `MemoryDenyWriteExecute`, `RestrictAddressFamilies`, etc.).
- **Supply-chain integrity.** Releases ship `sha256sums.txt` plus a SLSA build-provenance
  attestation (GitHub OIDC), built reproducibly (`-trimpath`, pinned `go.sum`).

The findings below are predominantly **hardening / defence-in-depth** items and **process**
observations rather than exploitable defects. No high or critical issues were identified.

## 2. Scope & methodology

**In scope:** all first-party Go packages (`cmd/syscert`, `internal/*`), the dependency graph,
and the released linux/amd64+arm64 binary and its build pipeline.

| Activity | Tool / method |
|---|---|
| Static security analysis | `gosec` (`./cmd/... ./internal/...`) |
| Vulnerable-dependency scan | `govulncheck` (reachable-symbol) |
| Correctness static analysis | `go vet` |
| Binary hardening inspection | `file`, `readelf -h/-d/-l`, `go version -m` |
| Build-pipeline review | `.github/workflows/release.yml` |
| Manual code review | secrets, crypto, file modes, privilege, input handling, `exec` use |

**Out of scope:** the Ansible role (separate repo); the operator's host hardening, disk
encryption, backup security, DNS-provider account security, and CA server security; penetration
testing / dynamic analysis against a live CA.

## 3. Architecture & trust boundaries

```
            (operator, root)                         (CA: LE / Vault / step-ca)
   writes /etc/syscert/{syscert.toml 0640, secrets 0640}        ▲  ACME over TLS (Go verifies
                     │                                          │  against system trust, or a
                     ▼                                          │  connection-only ca_bundle)
        ┌───────────────────────────┐   env: SYSCERT_EAB_HMAC,  │
        │  syscert (non-root user)  │───  DNS_* creds  ─────────┘
        │  systemd oneshot + timer  │
        │  CAP_CHOWN only           │
        └───────────────────────────┘
              │ writes 0600 keys              │ copies artefacts (owner/mode/SELinux)
              ▼                                ▼
   /var/lib/syscert  (0700 syscert)     consumer paths (e.g. /etc/nginx/tls/*)
   - account.key 0600                   - key-bearing artefacts forced non-world-readable
   - privkey.pem 0600
   - archive/<UTC>/ (optional history)
```

**Trust boundaries.** (a) Operator-supplied config/secrets are **trusted input** — written by
root or the syscert user, validated fail-fast. (b) The CA is **TLS-authenticated** (system trust,
or an explicit connection-only `ca_bundle` to bootstrap an internal CA). (c) DNS-provider APIs are
reached with operator-supplied credentials via lego. (d) Local consumers receive files but
SysCert **never executes consumer code or reload hooks**.

## 4. Automated analysis results

| Check | Result |
|---|---|
| `go vet ./...` | **Clean** (rc 0) |
| `gosec ./cmd/... ./internal/...` | **0 issues**, 26 files / 2,773 LOC, 9 reviewed `#nosec` |
| `govulncheck ./...` | **No vulnerabilities found** (rc 0); 676 modules, 0 reachable |
| Unit tests | Green, incl. security regressions (store-ownership preflight, status-never-leaks-HMAC, store modes) |
| Binary: linkage | **Static** (`CGO_ENABLED=0`), no `NEEDED`/`RUNPATH` |
| Binary: symbols | **Stripped** (`-s -w`), `-trimpath` |
| Binary: NX stack | **Yes** (`GNU_STACK … RW`) |
| Binary: PIE/ASLR | **No** — `Type: EXEC` (Go `-buildmode=exe` default) → Finding F-02 |
| Binary: provenance | `vcs.revision`/`vcs.time` embedded, `vcs.modified=false` |
| Release integrity | `sha256sums.txt` + SLSA build-provenance attestation (OIDC) |

**`#nosec` suppressions reviewed (all justified):**

- `G304` (file inclusion) ×5 — reads of syscert-owned store paths or operator-supplied
  `--ca-file`/`--env-file`/`ca_bundle` paths (trusted input by design).
- `G204` (subprocess) ×2 — the OS-detected trust-store update command and a fixed
  `chcon -t <ctx> <path>`; no shell, arguments are not attacker-controlled.
- `G306` (file perms) ×1 — CA trust anchors written `0644` **intentionally** (public certs must
  be world-readable in the system trust store).

## 5. Security controls — by domain

| Domain | Control in place | Status |
|---|---|---|
| **Memory safety** | Pure Go, GC, bounds-checked; `CGO_ENABLED=0` (no C) | ✅ Strong |
| **Secrets** | Credentials + EAB HMAC from env / `0640` file; never in TOML; never logged or printed (test-enforced); parse errors cite line numbers only | ✅ Strong |
| **Private key protection** | `account.key`/`privkey.pem` `0600`; bundle-with-key `0600`; key-bearing distribute targets rejected if world-readable | ✅ Strong |
| **Key management** | ECDSA P-256 default; **fresh keypair every renewal** (`reuse_key` opt-in); `crypto/rand` | ✅ Strong |
| **Privilege model** | Dedicated non-root `syscert` user; `CAP_CHOWN` only; refuses to run when the store is owned by another user (incl. root over a syscert store) | ✅ Strong |
| **Process isolation** | systemd: `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`, `PrivateTmp`, `Protect*`, `RestrictNamespaces/Realtime/SUIDSGID`, `LockPersonality`, `MemoryDenyWriteExecute`, `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX`, `CapabilityBoundingSet=CAP_CHOWN` | ✅ Strong |
| **SELinux** | Binary relabelled `bin_t`; per-target `selinux_context` via `chcon`; enforcing-mode supported | ✅ Good |
| **Input validation** | Fail-fast `validate.Config`; CA capability checks at runtime; `dry-run` surface; no FQDN ⇒ hard error | ✅ Strong |
| **Network / TLS** | Go stdlib `crypto/tls`; CA verified against system trust; connection-only `ca_bundle` is opt-in and warned | ✅ Good |
| **Command execution** | Only two `exec.Command` sites (trust-store update, `chcon`); fixed argv, no shell; no consumer reload hooks ever | ✅ Good |
| **Supply chain** | 2 direct deps; `go.sum` pinned; `govulncheck` gate; SLSA provenance + checksums | ⚠️ F-01 |
| **Build integrity** | `CGO_ENABLED=0 -trimpath -ldflags "-s -w"`, provenance attestation | ⚠️ F-02, F-05 |
| **Logging** | stderr→journal; secrets never logged; `status` is read-only and never prints the HMAC | ✅ Strong |

## 6. Findings

Each maps to a risk-register row (§7). All are Low or Informational.

- **F-01 — Large transitive dependency tree (676 modules via lego) — Low.** `go-acme/lego/v5`
  vendors every DNS-provider SDK. *Mitigated by* `go.sum` pinning, the mandatory `govulncheck`
  gate (0 reachable vulns), and that only the configured provider's code path runs. *Recommend*
  Dependabot/renovate for `lego` and keeping the gate blocking.
- **F-02 — Release binary is not position-independent (no full-binary ASLR) — Low.** `Type: EXEC`
  (Go `-buildmode=exe` default). NX/stack/heap ASLR still apply, and Go memory safety removes most
  exploit primitives. *Recommend* optionally building `-buildmode=pie` for defence-in-depth.
- **F-03 — Sensitive material persists on disk — Low.** Private keys (`0600`), secrets
  (`0640 root:syscert`); `archive_keep > 0` retains historical keys. *Mitigated by* tight modes
  and `0700` store; history off by default. *Recommend* protected storage / encrypted or excluded
  backups and a modest `archive_keep` (operator responsibility).
- **F-04 — `CAP_CHOWN` grants broad ownership-change ability — Low.** Needed to chown distributed
  artefacts. *Mitigated by* `CapabilityBoundingSet=CAP_CHOWN`, `NoNewPrivileges`,
  `ProtectSystem=strict`, bounded `ReadWritePaths`.
- **F-05 — No detached cryptographic signature on the binary — Low.** Integrity/origin via
  `sha256sums.txt` + SLSA provenance attestation (verify with `gh attestation verify`). *Recommend*
  optionally adding cosign/Sigstore signatures if a detached signature is required downstream.
- **F-06 — Connection-only `ca_bundle` can trust an arbitrary CA — Informational.** Opt-in,
  runtime-warned, connection-scoped (not the system trust store). Accept as documented.
- **F-07 — Single-maintainer, pre-1.0 project — Informational.** Mitigated by a disciplined,
  gated release process and an ADR log. *Recommend* keeping the disclosure policy current.

## 7. Risk register

Likelihood (L) / Impact (I) / Residual: **L**ow / **M**edium / **H**igh. Inherent = pre-control.

| ID | Risk | Category | L | I | Inherent | Key controls / mitigations | Residual | Status |
|---|---|---|---|---|---|---|---|---|
| R-01 | Vulnerable/malicious transitive dependency (676 modules) | Supply chain | M | M | Medium | 2 direct deps; `go.sum` pinning; mandatory `govulncheck` gate (0 reachable); only configured provider runs | **Low** | Open — monitored |
| R-02 | Keys / EAB HMAC / DNS creds exposed via host or backup compromise | Data protection | L | H | Medium | `0600` keys, `0640` secrets, `0700` store; never logged; history off by default | **Low** | Open — documented |
| R-03 | Secret leakage via logs/output | Data protection | L | H | Medium | Secrets only from env/file; never logged/printed; HMAC-absence test; parse errors cite line numbers only | **Low** | Closed — test-enforced |
| R-04 | Privilege escalation from the syscert service | Privilege/isolation | L | M | Medium | Non-root user; `CAP_CHOWN` only + bounding set; `NoNewPrivileges`; `ProtectSystem=strict`; `MemoryDenyWriteExecute`; store preflight | **Low** | Open — accepted |
| R-05 | Root run mis-owns the store, breaking renewals | Operational/integrity | L | M | Medium | Preflight refuses root-over-syscert-store and non-owner runs with guidance (v0.3.1) | **Low** | Closed — guarded |
| R-06 | Memory-safety exploit in the binary | Application | L | H | Medium | Pure Go (GC, bounds-checked); `CGO_ENABLED=0`; static; NX stack | **Low** | Closed — by design |
| R-07 | Tampered/substituted release artefact | Supply chain | L | H | Medium | `sha256sums.txt` + SLSA build-provenance attestation (OIDC); reproducible `-trimpath` build | **Low** | Open — see F-05 |
| R-08 | Command injection via subprocess execution | Application | L | M | Low | Only 2 `exec.Command` sites; fixed argv, no shell; never runs consumer hooks | **Low** | Closed |
| R-09 | MITM / wrong-CA issuance via misused `ca_bundle` | Network/trust | L | M | Low | Opt-in + runtime warning; connection-scoped; operator-set directory URL | **Low** | Closed — documented |
| R-10 | Invalid/unsafe configuration reaches issuance | Application | L | M | Low | Fail-fast `validate.Config`; runtime CA-capability checks; `dry-run`; hard error on missing FQDN | **Low** | Closed |
| R-11 | Exploitation aided by absent binary ASLR (non-PIE) | Application hardening | L | L | Low | Go memory safety; NX; short-lived non-network CLI | **Low** | Open — see F-02 |
| R-12 | Reduced assurance from single-maintainer / pre-1.0 | Governance | M | L | Low | Gated `prerelease.sh` (test/lint/vuln/gosec/provenance); ADR log; security policy | **Low** | Open — monitored |

**Residual risk profile:** all rows reduce to **Low**. No High/Critical residual risks.

## 8. Control framework mapping (indicative)

| Framework | Relevant area | Status |
|---|---|---|
| **OWASP ASVS** | V6 crypto (P-256 + fresh keys + `crypto/rand`); V2/V8 secrets (env/file, never logged) | Met |
| **CIS / least privilege** | Dedicated user, single capability, systemd sandboxing | Met |
| **SLSA** | Build provenance attestation (OIDC), scripted build, source-tracked (`vcs.modified=false`) | ~ Level 2–3 |
| **NIST SSDF** | PW.7 (review), PW.8 (test), RV.1 (`govulncheck`), PS.2 (provenance/checksums) | Met |
| **CWE hygiene** | No injection (CWE-78), safe file perms (CWE-276/732), no hardcoded secrets (CWE-798), memory-safe (CWE-119/416) | Clean |

## 9. Recommendations

1. **Keep the release gates mandatory** — `govulncheck` + `gosec` + tests in `prerelease.sh` stay
   blocking (R-01, R-03, R-06).
2. **Document operator data-protection prerequisites** — protected storage / encrypted backups for
   `/var/lib/syscert` and `/etc/syscert/secrets`; keep `archive_keep` modest (R-02, F-03).
3. **Publish the artefact-verification path** — `gh attestation verify` + `sha256sum --check`;
   optionally add cosign signatures (R-07, F-05).
4. **(Optional) PIE build** — evaluate `-buildmode=pie` (F-02, R-11).
5. **(Optional) Dependency hygiene** — Dependabot/renovate for `lego`; track advisories (R-01).

## 10. Conclusion

SysCert presents a **low overall security risk**. Memory-safe Go with no CGO, a dedicated
least-privilege user, an extensively sandboxed systemd unit, secrets that never touch the config or
logs, fresh keys per renewal, fail-fast validation, and provenance-attested reproducible releases
reflect a security-by-default posture. Automated analysis (`go vet`, `gosec`, `govulncheck`) is
clean. Open items are defence-in-depth and operator/process recommendations, none rated above Low
residual risk.

---

### Appendix A — reproduce this assessment

```sh
go vet ./...
gosec ./cmd/... ./internal/...                                  # 0 issues; 9 reviewed #nosec
go run golang.org/x/vuln/cmd/govulncheck@latest ./...           # No vulnerabilities found
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o syscert ./cmd/syscert
file syscert; readelf -h syscert | grep Type; readelf -l syscert | grep -A1 GNU_STACK
go version -m syscert | grep build
gh attestation verify syscert-linux-amd64 --repo tfindley/syscert   # release provenance
```

### Appendix B — assessment status

Produced by the SysCert maintainer using the tooling and methodology in §2, and published for
transparency. It is re-run as part of the release process (`prerelease.sh` gates `gosec`,
`govulncheck`, and the test suite on every release); findings and residual risks are reviewed at
each release, and material changes are reflected here and in the
[changelog](/changelog/). To report a vulnerability, see the
[Security policy](https://github.com/tfindley/syscert/blob/main/SECURITY.md).
