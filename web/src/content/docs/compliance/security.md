---
title: Security assessment
navLabel: Security
description: SysCert's published security assessment and risk register — static analysis (go vet, gosec, govulncheck), binary hardening, controls by domain, findings, residual-risk register, and control-framework mapping.
order: 1
eyebrow: "// docs · compliance · security"
lede: A published, tool-backed security assessment of the SysCert Go application and release binary — with a full risk register, so you can review the posture before you deploy.
---

This assessment is published for transparency. It covers the **Go application and the released binary**; it is re-run as part of every release (`prerelease.sh` gates `gosec`, `govulncheck`, and the test suite). To report a vulnerability, see the [Security policy](https://github.com/tfindley/syscert/blob/main/SECURITY.md).

| | |
|---|---|
| **Version assessed** | v0.5.0 (git `5c19edf`, `vcs.modified=false`) |
| **Assessment date** | 2026-09-03 |
| **Method** | Tool-assisted static analysis + manual review (see §2) |
| **Distribution** | Public |
| **Overall posture** | **Low risk.** Memory-safe Go, no CGO, least-privilege runtime, secrets never logged, fail-fast validation, signed-provenance releases. Zero findings from `go vet`, `gosec`, and `govulncheck`. |

> Scope note: the Ansible role ships in-tree at `packaging/ansible/` and is **in scope** for
> the deployment-surface review in §5 only — it is configuration management, not part of the
> audited Go binary, and it is not covered by the static analysis above. `install.sh`/
> `net-install.sh`/`offline-bundle.sh` (shell) and the systemd units are covered where they
> define the binary's runtime privilege/permissions context, and — since v0.5.0 — where the
> installer grants filesystem access on the service's behalf (§5, F-08).

## 1. Executive summary

SysCert is a small, single-purpose Go program that obtains an ACME/TLS certificate for a host (Let's Encrypt, HashiCorp Vault PKI, or Smallstep step-ca), stores it under a locked syscert-owned directory, and copies the requested artefacts to local consumers with explicit ownership/mode/SELinux context. It runs **as a dedicated non-root system user on a systemd timer**, with no long-running daemon and no inbound network surface in the default (dns-01) configuration.

The security posture is **strong for its size and stage**:

- **Automated analysis is clean.** `go vet` (clean), `gosec` (**0 issues**, 9 reviewed `#nosec` suppressions), and `govulncheck` (**0 vulnerabilities** across 676 modules).
- **Memory-safe, statically linked.** Pure Go, `CGO_ENABLED=0` → no C/libc memory-safety class of defects and no shared-library hijack surface.
- **Secrets are handled correctly.** DNS/CA credentials and the EAB HMAC come from the environment / a `0640` file, are **never written to the config, logged, or printed**, and this is enforced by a regression test.
- **Least privilege at runtime.** Dedicated `syscert` user, a single ambient capability (`CAP_CHOWN`), and an extensively hardened systemd unit (`ProtectSystem=strict`, `MemoryDenyWriteExecute`, `RestrictAddressFamilies`, etc.).
- **Supply-chain integrity.** Releases ship `sha256sums.txt` plus a SLSA build-provenance attestation (GitHub OIDC), built reproducibly (`-trimpath`, pinned `go.sum`). Every install route — `install.sh`, `net-install.sh`, `offline-bundle.sh` and the Ansible role — verifies the checksum with **no opt-out**.

The one place the surface **grew** in v0.5.0 is deployment, not the binary: to deliver into directories other packages own, the **installer** now applies a POSIX ACL per configured target directory and installs a systemd drop-in widening `ReadWritePaths`. Both are derived from the operator's own config, logged per directory, and reversed on uninstall — but they are a real, new install-time privilege-adjacent action, and they are assessed as such in §5 and F-08/F-09.

The findings below are predominantly **hardening / defence-in-depth** items and **process** observations rather than exploitable defects. No high or critical issues were identified.

## 2. Scope & methodology

**In scope:** all first-party Go packages (`cmd/syscert`, `internal/*`), the dependency graph, and the released linux/amd64+arm64 binary and its build pipeline.

| Activity | Tool / method |
|---|---|
| Static security analysis | `gosec` (`./cmd/... ./internal/...`) |
| Vulnerable-dependency scan | `govulncheck` (reachable-symbol) |
| Correctness static analysis | `go vet` |
| Binary hardening inspection | `file`, `readelf -h/-d/-l`, `go version -m` |
| Build-pipeline review | `.github/workflows/release.yml` |
| Manual code review | secrets, crypto, file modes, privilege, input handling, `exec` use |

**Out of scope:** the operator's host hardening, disk encryption, backup security, DNS-provider account security, and CA server security; penetration testing / dynamic analysis against a live CA. The in-tree Ansible role is reviewed as a deployment surface in §5, not as part of the binary's static analysis.

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

**Trust boundaries.** (a) Operator-supplied config/secrets are **trusted input**, written by root or the syscert user and validated fail-fast. (b) The CA is **TLS-authenticated** (system trust, or an explicit connection-only `ca_bundle` to bootstrap an internal CA). (c) DNS-provider APIs are reached with operator-supplied credentials via lego. (d) Local consumers receive files but SysCert **never executes consumer code or reload hooks**.

## 4. Automated analysis results

| Check | Result |
|---|---|
| `go vet ./...` | **Clean** (rc 0) |
| `gosec ./cmd/... ./internal/...` | **0 issues**, 31 non-test files / ~3,485 LOC, 11 reviewed `#nosec` |
| `govulncheck ./...` | **No vulnerabilities found** (rc 0); 676 modules, 0 reachable |
| Unit tests | Green, incl. security regressions (store-ownership preflight, status-never-leaks-HMAC, store modes) |
| Binary: linkage | **Static** (`CGO_ENABLED=0`), no `NEEDED`/`RUNPATH` |
| Binary: symbols | **Stripped** (`-s -w`), `-trimpath` |
| Binary: NX stack | **Yes** (`GNU_STACK … RW`) |
| Binary: PIE/ASLR | **No** — `Type: EXEC` (Go `-buildmode=exe` default) → Finding F-02 |
| Binary: provenance | `vcs.revision`/`vcs.time` embedded, `vcs.modified=false` |
| Release integrity | `sha256sums.txt` + SLSA build-provenance attestation (OIDC) |

**`#nosec` suppressions reviewed (all 11 justified):**

- `G304` (file inclusion) ×5 — reads of syscert-owned store paths or operator-supplied `--ca-file`/`--env-file`/`ca_bundle` paths (trusted input by design).
- `G204` (subprocess) ×2 — the OS-detected trust-store update command and a fixed `chcon -t <ctx> <path>`; no shell, arguments are not attacker-controlled.
- `G306`/`G301` (file and directory perms) ×4 — CA trust anchors written `0644` (public certs *must* be world-readable in the system trust store), and the `systemd-paths` drop-in file (`0644`) and its directory (`0755`), which systemd must be able to read and traverse.

The v0.5.0 observability outputs need no suppression: they are written `0644` through `internal/atomicfile`, which creates the temp file `0600` and `chmod`s it only after the contents are written, so no file ever exists at a wider mode than intended.

## 5. Security controls — by domain

| Domain | Control in place | Status |
|---|---|---|
| **Memory safety** | Pure Go, GC, bounds-checked; `CGO_ENABLED=0` (no C) | ✅ Strong |
| **Secrets** | Credentials + EAB HMAC from env / `0640` file; never in TOML; never logged or printed (test-enforced); parse errors cite line numbers only | ✅ Strong |
| **Private key protection** | `account.key`/`privkey.pem` `0600`; bundle-with-key `0600`; key-bearing distribute targets rejected if world-readable | ✅ Strong |
| **Key management** | ECDSA P-256 default; **fresh keypair every renewal** (unconditionally — `reuse_key` is accepted but not yet applied); `crypto/rand` | ✅ Strong |
| **Privilege model** | Dedicated non-root `syscert` user; `CAP_CHOWN` only; refuses to run when the store is owned by another user (incl. root over a syscert store) | ✅ Strong |
| **Process isolation** | systemd: `NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`, `PrivateTmp`, `Protect*`, `RestrictNamespaces/Realtime/SUIDSGID`, `LockPersonality`, `MemoryDenyWriteExecute`, `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX`, `CapabilityBoundingSet=CAP_CHOWN` | ✅ Strong |
| **SELinux** | Binary relabelled `bin_t`; per-target `selinux_context` via `chcon`; enforcing-mode supported | ✅ Good |
| **Input validation** | Fail-fast `validate.Config`; CA capability checks at runtime; `dry-run` surface; no FQDN ⇒ hard error | ✅ Strong |
| **Network / TLS** | Go stdlib `crypto/tls`; CA verified against system trust; connection-only `ca_bundle` is opt-in and warned | ✅ Good |
| **Command execution** | Only two `exec.Command` sites (trust-store update, `chcon`); fixed argv, no shell; no consumer reload hooks ever. Re-verified at v0.5.0: `systemd-paths --write` **prints** `systemctl daemon-reload` rather than running it, so the count is unchanged | ✅ Good |
| **Observability outputs (`[observe]`)** | Opt-in, off unless a path is set; write-only and never read back, so they cannot influence behaviour; atomic write at `0644` (node_exporter and Ansible are not the syscert user). The snapshot carries only certificate metadata — subject, CA *name*, challenge, issuer, serial, validity/renewal times, key type, and target paths/presence. No credential, EAB HMAC, directory URL or key material is in the struct or its populating code | ✅ Good — F-10 |
| **Supply chain** | 2 direct deps; `go.sum` pinned; `govulncheck` gate; SLSA provenance + checksums | ⚠️ F-01 |
| **Build integrity** | `CGO_ENABLED=0 -trimpath -ldflags "-s -w"`, provenance attestation | ⚠️ F-02, F-05 |
| **Logging** | stderr→journal; secrets never logged; `status` is read-only and never prints the HMAC | ✅ Strong |
| **Fleet deployment (Ansible role)** | In-tree at `packaging/ansible/`; release binary always checksum-verified against `sha256sums.txt` with no opt-out; secrets rendered `0640 root:syscert` under `no_log`; renders the same hardened unit and derives `ReadWritePaths` from the declared distribute targets; validates the rendered config with `dry-run --config-only` before enabling the timer; `ansible-lint` (production profile) gated in CI | ⚠️ See note |

### 5.1 The installer's ACL grant — what it actually gives away

This is the largest new privilege-adjacent behaviour in v0.5.0 and is stated in full rather than summarised. To deliver into a directory another package owns, both `install.sh` and the Ansible role apply a POSIX ACL (`setfacl -m u:syscert:rwx <dir>`) to each configured `[[distribute]]` target directory and each `[observe]` output directory, and install a systemd drop-in naming those same directories in `ReadWritePaths`.

Why an ACL: the alternatives all trade a large permanent privilege for a small local one. `CAP_DAC_OVERRIDE` would let the service bypass **every** file permission check host-wide to fix one directory; relaxing `ProtectSystem=strict` would surrender the sandbox ADR-0028 exists for. An ACL is one user on one directory, and unlike `chgrp`+`g+w` it leaves owner and group exactly as the owning package set them, so `rpm -V` and the package's own expectations are undisturbed (ADR-0048).

What it grants, stated plainly:

- **Directory write is directory-wide.** `rwx` on a directory lets the syscert user create, rename and **unlink** *any* file in it — not only the artefacts syscert placed there. A target in `/etc/nginx/tls` therefore means the syscert user can replace or delete everything in `/etc/nginx/tls`. This is inherent to writing a file there at all; it is not narrowed by the ACL.
- **The depth guard is a typo guard, not a security boundary.** Both routes refuse anything shallower than two path components (`^(/[^/]+){2,}/?$`) — so a mistyped `path = "/etc/x.pem"` cannot grant `/etc`. It does *not* refuse a two-component directory that happens to be sensitive; `/etc/sudoers.d`, `/etc/cron.d` and `/usr/bin` all satisfy the pattern. Pointing a distribute target at one of those would hand the service user a root-escalation primitive. The config is root-authored trusted input, so this is an operator-error class, not an attacker path — but it is the reason target paths deserve review, and it is tracked as F-08.
- **The grants are logged and reversible, but revocation is config-derived.** `--uninstall` (and the role's uninstall tasks) re-derive the directory list from the *current* config and strip the ACL from each. A directory that was granted and later removed from the config is therefore never revoked; after `--purge` deletes the user, that ACL persists as a bare numeric uid. Tracked as F-08.
- **The binary changes no permissions.** Only the installer does. `systemd-paths --write` writes a drop-in and prints `systemctl daemon-reload` rather than running it, so the two-`exec.Command`-site property above holds — independently re-verified at this commit.

### 5.2 Deployment-surface notes

The Ansible role fetches the binary and its checksum file from the same origin (`syscert_download_base_url`), so verification proves integrity, not provenance — an operator pointing it at a hostile mirror gets a matching pair. The same is true of `install.sh`/`net-install.sh`. The unattended-install path is therefore only as trustworthy as that origin; `syscert_install_method: local` (stage and verify on the controller) is the stronger route for high-assurance fleets, and `scripts/offline-bundle.sh` additionally runs `gh attestation verify` when `gh` is present — best-effort, so an operator who needs a hard provenance gate should verify the attestation explicitly before staging (F-05). The role does not run `syscert trust install`, so internal-CA roots remain a documented manual step.

The role creates only directories that are **missing** (root-owned `0755`); it never re-owns or re-modes an existing one, so a node_exporter- or package-owned directory keeps the ownership its owner set. `install.sh` is stricter still — it warns and skips a directory that does not exist.

## 6. Findings

Each maps to a risk-register row (§7). All are Low or Informational.

- **F-01 — Large transitive dependency tree (676 modules via lego) — Low.** `go-acme/lego/v5` vendors every DNS-provider SDK. *Mitigated by* `go.sum` pinning, the mandatory `govulncheck` gate (0 reachable vulns), and that only the configured provider's code path runs. *Recommend* Dependabot/renovate for `lego` and keeping the gate blocking.
- **F-02 — Release binary is not position-independent (no full-binary ASLR) — Low.** `Type: EXEC` (Go `-buildmode=exe` default). NX/stack/heap ASLR still apply, and Go memory safety removes most exploit primitives. *Recommend* optionally building `-buildmode=pie` for defence-in-depth.
- **F-03 — Sensitive material persists on disk — Low.** Private keys (`0600`), secrets (`0640 root:syscert`); `archive_keep > 0` retains historical keys. *Mitigated by* tight modes and `0700` store; history off by default. *Recommend* protected storage / encrypted or excluded backups and a modest `archive_keep` (operator responsibility).
- **F-04 — `CAP_CHOWN` grants broad ownership-change ability — Low.** Needed to chown distributed artefacts. *Mitigated by* `CapabilityBoundingSet=CAP_CHOWN`, `NoNewPrivileges`, `ProtectSystem=strict`, bounded `ReadWritePaths`.
- **F-05 — No detached cryptographic signature on the binary — Low.** Integrity/origin via `sha256sums.txt` + SLSA provenance attestation (verify with `gh attestation verify`). *Recommend* optionally adding cosign/Sigstore signatures if a detached signature is required downstream.
- **F-06 — Connection-only `ca_bundle` can trust an arbitrary CA — Informational.** Opt-in, runtime-warned, connection-scoped (not the system trust store). Accept as documented.
- **F-07 — Single-maintainer, pre-1.0 project — Informational.** Mitigated by a disciplined, gated release process and an ADR log. *Recommend* keeping the disclosure policy current.
- **F-08 — Installer ACL grants are directory-wide and revoked only from the current config — Low.** *(new in v0.5.0.)* `setfacl -m u:syscert:rwx <dir>` lets the service user create, rename and unlink every file in that directory, and the two-component depth guard rejects only top-level paths — a target under `/etc/sudoers.d` or `/usr/bin` would pass it. Removing a target from the config leaves its ACL in place, and `--purge` then deletes the user, orphaning the entry as a numeric uid. *Mitigated by* the config being root-authored trusted input, per-directory logging at install time, owner/group left untouched, and a grant list derived from the operator's own config rather than a wider static default. *Recommend* reviewing distribute target paths as part of change control, treating them as privileged; and re-running the installer (not just editing the config) when targets change, so the grant set tracks it. See §5.1.
- **F-09 — Directory strings reach generated systemd fragments unvalidated — Low.** *(new in v0.5.0.)* `dropInContent()` and the role's `syscert.service.j2` render each directory into `ReadWritePaths=` verbatim. A `[[distribute]] path` that is relative, contains whitespace, or contains a newline therefore produces a malformed or over-broad grant — a newline places an arbitrary line into the drop-in's `[Service]` section. `internal/validate` checks the `[observe]` paths are absolute but applies no equivalent rule to `distribute[].path`. **No privilege boundary is crossed**: the config is `0640 root:syscert` and the drop-in writer requires root, so an actor who can trigger this can already write the unit. *Recommend* rejecting non-absolute directories, and directories containing whitespace or control characters, in `outputDirs()`/validation, and quoting the rendered paths — matching the newline rule the Ansible role already enforces on secret values.
- **F-10 — Observability outputs publish operational metadata world-readable — Informational.** *(new in v0.5.0.)* `metrics_file` and `ansible_facts_file` are written `0644` by design, because node_exporter and Ansible are not the syscert user. They contain no secret — the snapshot is certificate metadata plus target paths — but they do disclose the host's subject/SANs posture, issuer, serial, expiry, and the on-disk locations of key-bearing artefacts to every local user. Both are **off by default**. Accept as documented; operators for whom local metadata disclosure matters should leave them unset or place them in a mode-restricted directory.

## 7. Risk register

Likelihood (L) / Impact (I) / Residual: **L**ow / **M**edium / **H**igh. Inherent = pre-control.

| ID | Risk | Category | L | I | Inherent | Key controls / mitigations | Residual | Status |
|---|---|---|---|---|---|---|---|---|
| R-01 | Vulnerable/malicious transitive dependency (676 modules) | Supply chain | M | M | Medium | 2 direct deps; `go.sum` pinning; mandatory `govulncheck` gate (0 reachable); only configured provider runs | **Low** | Open — monitored |
| R-02 | Keys / EAB HMAC / DNS creds exposed via host or backup compromise | Data protection | L | H | Medium | `0600` keys, `0640` secrets, `0700` store; never logged; history off by default | **Low** | Open — documented |
| R-03 | Secret leakage via logs/output | Data protection | L | H | Medium | Secrets only from env/file; never logged/printed; HMAC-absence test; parse errors cite line numbers only | **Low** | Closed — test-enforced |
| R-04 | Privilege escalation from the syscert service | Privilege/isolation | L | M | Medium | Non-root user; `CAP_CHOWN` only + bounding set; `NoNewPrivileges`; `ProtectSystem=strict`; `MemoryDenyWriteExecute`; store preflight. ACL grants are per-directory and config-derived, never `CAP_DAC_OVERRIDE` | **Low** | Open — accepted; re-rated at v0.5.0, see R-13 |
| R-05 | Root run mis-owns the store, breaking renewals | Operational/integrity | L | M | Medium | Preflight refuses root-over-syscert-store and non-owner runs with guidance (v0.3.1) | **Low** | Closed — guarded |
| R-06 | Memory-safety exploit in the binary | Application | L | H | Medium | Pure Go (GC, bounds-checked); `CGO_ENABLED=0`; static; NX stack | **Low** | Closed — by design |
| R-07 | Tampered/substituted release artefact | Supply chain | L | H | Medium | `sha256sums.txt` + SLSA build-provenance attestation (OIDC); reproducible `-trimpath` build | **Low** | Open — see F-05 |
| R-08 | Command injection via subprocess execution | Application | L | M | Low | Only 2 `exec.Command` sites; fixed argv, no shell; never runs consumer hooks | **Low** | Closed |
| R-09 | MITM / wrong-CA issuance via misused `ca_bundle` | Network/trust | L | M | Low | Opt-in + runtime warning; connection-scoped; operator-set directory URL | **Low** | Closed — documented |
| R-10 | Invalid/unsafe configuration reaches issuance | Application | L | M | Low | Fail-fast `validate.Config`; runtime CA-capability checks; `dry-run`; hard error on missing FQDN | **Low** | Closed |
| R-11 | Exploitation aided by absent binary ASLR (non-PIE) | Application hardening | L | L | Low | Go memory safety; NX; short-lived non-network CLI | **Low** | Open — see F-02 |
| R-12 | Reduced assurance from single-maintainer / pre-1.0 | Governance | M | L | Low | Gated `prerelease.sh` (test/lint/vuln/gosec/provenance); ADR log; security policy | **Low** | Open — monitored |
| R-13 | Over-broad installer ACL grant from a mis-set distribute target (e.g. `/etc/sudoers.d`, `/usr/bin`) | Privilege/isolation | L | H | Medium | Two-component depth guard on both install routes; grant derived from the operator's own config; logged per directory; owner/group untouched; config is root-authored trusted input; role hard-fails on an ungrantable path | **Low** | Open — new in v0.5.0 (F-08) |
| R-14 | Malformed or injected `ReadWritePaths` entry from an unvalidated distribute path | Application hardening | L | M | Low | Config is `0640 root:syscert`; drop-in writer is root-only, so no boundary is crossed; installers apply their own depth guard before granting | **Low** | Open — new in v0.5.0 (F-09) |
| R-15 | Local disclosure of certificate/inventory metadata via the `[observe]` files | Data protection | L | L | Low | Off by default; contains no key, credential or EAB material (struct-level verified); `0644` is a deliberate requirement of node_exporter/Ansible | **Low** | Open — accepted (F-10) |
| R-16 | Stale ACL survives a target change or uninstall, then a reused uid inherits it | Privilege/isolation | L | M | Low | Revocation re-derives the list from the config and runs before the user is deleted; system uids are not recycled by default on Debian/RHEL | **Low** | Open — new in v0.5.0 (F-08) |

**Residual risk profile:** all rows reduce to **Low**. No High/Critical residual risks. The v0.5.0 additions (R-13 – R-16) are all deployment-surface rows, all conditional on operator configuration, and none change the binary's own posture.

## 8. Control framework mapping (indicative)

| Framework | Relevant area | Status |
|---|---|---|
| **OWASP ASVS** | V6 crypto (P-256 + fresh keys + `crypto/rand`); V2/V8 secrets (env/file, never logged) | Met |
| **CIS / least privilege** | Dedicated user, single capability, systemd sandboxing | Met |
| **SLSA** | Build provenance attestation (OIDC), scripted build, source-tracked (`vcs.modified=false`) | ~ Level 2–3 |
| **NIST SSDF** | PW.7 (review), PW.8 (test), RV.1 (`govulncheck`), PS.2 (provenance/checksums) | Met |
| **CWE hygiene** | No injection (CWE-78), safe file perms (CWE-276/732), no hardcoded secrets (CWE-798), memory-safe (CWE-119/416) | Clean |

## 9. Recommendations

1. **Keep the release gates mandatory** — `govulncheck` + `gosec` + tests in `prerelease.sh` stay blocking (R-01, R-03, R-06).
2. **Document operator data-protection prerequisites** — protected storage / encrypted backups for `/var/lib/syscert` and `/etc/syscert/secrets`; keep `archive_keep` modest (R-02, F-03).
3. **Publish the artefact-verification path** — `gh attestation verify` + `sha256sum --check`; optionally add cosign signatures (R-07, F-05).
4. **Validate directory strings before they reach a generated unit fragment** — reject non-absolute paths, and paths containing whitespace or control characters, in `outputDirs()`/`validate.Config`; quote what is rendered. The role already enforces exactly this rule on secret values (F-09, R-14).
5. **Treat distribute target paths as privileged configuration** — review them in change control, and re-run the installer when they change so the ACL set tracks the config (F-08, R-13, R-16).
6. **(Optional) PIE build** — evaluate `-buildmode=pie` (F-02, R-11).
7. **(Optional) Dependency hygiene** — Dependabot/renovate for `lego`; track advisories (R-01).

## 10. Conclusion

SysCert presents a **low overall security risk**. Memory-safe Go with no CGO, a dedicated least-privilege user, an extensively sandboxed systemd unit, secrets that never touch the config or logs, fresh keys per renewal, fail-fast validation, and provenance-attested reproducible releases reflect a security-by-default posture. Automated analysis (`go vet`, `gosec`, `govulncheck`) is clean. Open items are defence-in-depth and operator/process recommendations, none rated above Low residual risk.

At v0.5.0 the binary's own posture is unchanged — key handling, secret sourcing, trust and store permissions have no code changes in the release range — while the **deployment** surface grew: the installer now grants filesystem access on the service's behalf, and two new optional output files are written world-readable by design. Both are opt-in, both are derived from and bounded by the operator's own config, and both are assessed above rather than left implicit.

---

### Appendix A — reproduce this assessment

```sh
go vet ./...
gosec ./cmd/... ./internal/...                                  # 0 issues; 11 reviewed #nosec
go run golang.org/x/vuln/cmd/govulncheck@latest ./...           # No vulnerabilities found
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o syscert ./cmd/syscert
file syscert; readelf -h syscert | grep Type; readelf -l syscert | grep -A1 GNU_STACK
go version -m syscert | grep build
gh attestation verify syscert-linux-amd64 --repo tfindley/syscert   # release provenance

# §5 claims, independently checkable:
grep -rn 'exec\.Command' cmd internal        # exactly 2 sites (trust update, chcon)
grep -rn '#nosec' cmd internal | wc -l       # 11
./syscert systemd-paths --config <path>      # the drop-in, printed not written
```

### Appendix B — assessment status

Produced by the SysCert maintainer using the tooling and methodology in §2, and published for transparency. It is re-run as part of the release process (`prerelease.sh` gates `gosec`, `govulncheck`, and the test suite on every release); findings and residual risks are reviewed at each release, and material changes are reflected here and in the [changelog](/changelog/). To report a vulnerability, see the [Security policy](https://github.com/tfindley/syscert/blob/main/SECURITY.md).
