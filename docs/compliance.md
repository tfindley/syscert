---
title: Compliance
navLabel: Compliance
description: SysCert's compliance and assurance posture — the published security assessment and risk register, the release-time security gates, and supply-chain provenance.
order: 9
eyebrow: "// docs · compliance"
lede: How SysCert demonstrates its security posture — a published assessment, gated releases, and verifiable provenance you can hand to a review team.
---

SysCert is built to be reviewable. Below is the assurance evidence a security or compliance team
needs before signing off on production.

## What's here

- **[Security assessment](/docs/compliance/security/)** — a published, tool-backed assessment of
  the Go application and the release binary, with a full **risk register**, controls-by-domain,
  findings, and control-framework mapping (OWASP ASVS, CIS, SLSA, NIST SSDF, CWE).
- **[Tech stack](/docs/compliance/tech-stack/)** — what SysCert is built from: the static Go
  binary and its dependencies, the runtime model, the docs site, and the build/release pipeline.
- **[AI-assisted development](/docs/compliance/ai-assisted-development/)** — how SysCert is built
  (AI-assisted, human-directed) and the testing, gating, and review controls behind it.

## How assurance is maintained

Releases are gated. Every release runs `scripts/prerelease.sh`, which blocks on `go vet`, `gofmt`,
the test suite, `gosec`, and `govulncheck`; if any of those fail, the release doesn't ship. The
security assessment gets re-validated against each release too.

Every release binary ships `sha256sums.txt` plus a SLSA build-provenance attestation (GitHub OIDC),
built reproducibly with `CGO_ENABLED=0 -trimpath`. Verify a download with `sha256sum --check` and
`gh attestation verify <file> --repo tfindley/syscert`.

The binary is memory-safe by construction. It's pure Go with no CGO, which takes whole classes of
memory-safety bugs off the table.

## Reporting

Security vulnerabilities: see the
[Security policy](https://github.com/tfindley/syscert/blob/main/SECURITY.md).

---

Next: [Security assessment](/docs/compliance/security/) · [FAQ](/docs/faq/)
