---
title: AI-assisted development
navLabel: AI-assisted development
description: How SysCert is built — AI-assisted implementation under maintainer direction, with test-first development, blocking automated security gates, real-world testing, and human review and sign-off.
order: 3
eyebrow: "// docs · compliance · ai-assisted development"
lede: SysCert is developed with AI assistance — under human direction, validated by tests and automated security gates, and tested on real systems. Here's exactly how, and the controls that make it trustworthy.
---

Full transparency: **SysCert is developed with AI assistance**, mostly [Anthropic's Claude](https://www.anthropic.com/claude) via Claude Code, working under the maintainer's direction. This page explains how that works, and the controls that keep what ships **directed, tested, reviewed, and owned by a human** rather than unvalidated model output.

## The short version

AI accelerates *implementation*. Humans own *intent, validation, and release*. Every line that ships passes the same automated gates and human review regardless of how it was first drafted.

## How a change is made

1. **Scoped & planned by the maintainer.** Work starts from a requirement. An approach gets proposed, then **reviewed and approved before implementation** begins. The maintainer decides the design choices; nothing is assumed.
2. **Implemented test-first.** Tests (`go test`) cover the behaviour first, so changes are pinned by executable expectations instead of asserted to work.
3. **Statically analysed.** `go vet`, `gofmt`, **`gosec`** (security static analysis), and **`govulncheck`** (known-vulnerability scan of the dependency graph) run over the code.
4. **Human-reviewed.** Changes go through review passes (correctness and quality) and the maintainer's reading before they land.
5. **Gated at release.** `scripts/prerelease.sh` re-runs the full set (`go vet`, `gofmt`, `go test`, `gosec`, `govulncheck`, plus docs/CLI parity and config-validation checks), and a release **cannot be cut while any of them fail**.
6. **Tested on real systems.** The maintainer deploys SysCert against real **HashiCorp Vault PKI** and **RHEL/SELinux-enforcing** hosts. That testing has surfaced fixes no amount of code review would have caught: the SELinux binary-label, certificate-ownership, and unprivileged-access issues turned up in live deployments and were fixed in v0.3.1.
7. **Released with provenance.** Binaries are built reproducibly and ship a SLSA build-provenance attestation plus `sha256sums.txt` (see [Tech stack](/docs/compliance/tech-stack/)).

## Why this is trustworthy

The validation doesn't care who wrote the line. Those gates in steps 3–5 treat AI-drafted and hand-written code the same way: failing tests, a `gosec` finding, or a `govulncheck` vulnerability block the release either way.

And the evidence is public. The [Security assessment](/docs/compliance/security/) reports the actual tool results (`gosec`: 0 issues; `govulncheck`: 0 vulnerabilities) alongside a full risk register, re-validated each release.

Someone stays accountable for all of it. The maintainer directs the work, reviews it, tests it in production-like environments, and signs off on every release. AI assistance is a tool in that process, not a substitute for it.

## What this means for you

You can evaluate SysCert the way you'd evaluate any other dependency: read the code, run the same checks, verify the release provenance, and test it in your own environment (the exact commands are in the reproduce-it appendix of the [Security assessment](/docs/compliance/security/)). The method doesn't ask for your trust; the controls and published evidence are there to be checked.

---

Next: [Security assessment](/docs/compliance/security/) · [Tech stack](/docs/compliance/tech-stack/)
