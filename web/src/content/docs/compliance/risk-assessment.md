---
title: Risk assessment
navLabel: Risk
description: An impartial adoption and operational risk assessment of SysCert covering consumption, production operation, and supply chain. Written from a due-diligence perspective that treats the maintainer as an untrusted, single-person upstream you should be ready to fork, and mapped to the FitSD Service Acceptance Criteria.
order: 1.5
eyebrow: "// docs · compliance · risk"
lede: A due-diligence risk assessment for adopting and running SysCert. It's deliberately impartial, treating the project as untrusted, single-maintainer, pre-1.0 open source you should be prepared to fork, and mapping the residual risk to the FitSD Service Acceptance Criteria.
---

This is a risk assessment for **taking on and running** SysCert, written for whoever has to sign off
on adopting it. Think of it as a companion to the [Security assessment](/docs/compliance/security/)
rather than a copy: that document asks whether the *code* is secure, this one asks whether
*depending on the project and operating the service* is an acceptable risk, and what you have to own
to make it so.

It's written to be impartial, and the maintainer is not treated as a trusted supplier. There's no
company behind SysCert. No SLA, no support contract, no promise the project is still maintained next
year. Read every "SysCert does X" below as a claim you can check yourself against the source. You
can, and that's rather the point: you don't have to trust whoever wrote it.

| | |
|---|---|
| **Subject** | SysCert — ACME/TLS certificate lifecycle tool (single static Go binary + systemd timer) |
| **Licence** | AGPL-3.0-or-later (source guaranteed available; forkable) |
| **Assessment type** | Adoption + operational + supply-chain risk (service acceptance) |
| **Perspective** | Impartial / adopter due diligence — maintainer assumed **untrusted** |
| **Framework** | Mapped to the [FitSD Service Acceptance Criteria](https://fitsd.tfindley.dev) (FSD-PRO §7 / FSD-SD-5) |
| **Overall** | **Acceptable with owned mitigations.** Low technical risk. The residual risks are organisational: single-maintainer supplier viability, plus the operator-owned monitoring and backup. All acceptable *because* the software is forkable, self-buildable, and has no runtime dependency on the maintainer. |

## 1. The runtime has no dependency on the maintainer

Once it's installed, SysCert never contacts the maintainer or the project's website. It talks to one
external thing: the ACME directory URL *you* configure, whether that's Let's Encrypt, your internal
HashiCorp Vault, or step-ca. No telemetry, no update check, nothing phoning home. (Verified: the only
external hosts in the binary are the Let's Encrypt directory defaults and two documentation URLs that
sit in help text, not network calls.)

That one fact does most of the de-risking.

An abandoned upstream doesn't break a running deployment. If the maintainer disappears tomorrow,
every host already running SysCert keeps issuing and renewing against your CA, indefinitely. There's
nothing you need from upstream to keep the lights on. And the supplier sits in neither your data path
nor your trust path: your CA issues the certificates, your trust anchors verify them, and the
maintainer is upstream of your source code, not your runtime.

So maintainer risk here is really a change-management risk. Will there be future fixes and releases?
It isn't an availability risk. That's the distinction that makes a single-maintainer, pre-1.0 project
a defensible choice rather than a reckless one.

## 2. What you are actually depending on

| Dependency | What it is | If it goes bad | Your exit |
|---|---|---|---|
| The **binary** | One static, CGO-free executable | Nothing external can revoke or disable it | Keep a copy; rebuild from source |
| The **maintainer** | One person, upstreams fixes + releases | No new fixes/releases | Fork; self-maintain (AGPL guarantees the source) |
| The **install channel** | `syscert.tfindley.dev` + GitHub Releases | Unreachable / compromised | [Offline bundle](/docs/advanced-install/offline/) from a self-hosted mirror; build your own |
| The **CA** | Your Let's Encrypt / Vault / step-ca | Your problem, not SysCert's | Swap `directory_url`; standard ACME |
| The **dependencies** | 2 direct (`lego`, `toml`); large transitive tree via `lego` | Vulnerable/malicious dep | `go.sum` pinned; `govulncheck` gate; see [Security assessment R-01](/docs/compliance/security/) |

The pattern: everything you depend on is either already yours or something you can take over. The one
thing you can't fully control is the quality and pace of future upstream work, and the mitigation
there is that you're legally and technically free to do that work yourself.

## 3. Consumption / adoption risk

The risk of taking the dependency on in the first place.

**Supplier viability is the headline risk.** Bus factor of one, pre-1.0, no commercial entity behind
it. No paid support, no guaranteed issue response, no roadmap commitment. If you need a vendor you can
call at 3am, this isn't that, and no amount of code quality changes it. What makes it workable:
AGPL-3.0, two direct dependencies, a small first-party codebase. Forking and self-maintaining is
realistic rather than theoretical. Budget for the chance that you end up maintaining your own copy.

**Maturity and churn.** Pre-1.0 means the CLI flags, config schema, and defaults can still shift
between minor versions. You get warning, though. Releases follow SemVer and Conventional Commits, ship
a changelog, and are gated by `prerelease.sh` (tests, lint, `gosec`, `govulncheck`, provenance), with
design decisions recorded in an ADR log. Pin a version and read the changelog before you upgrade.

**Licence: AGPL-3.0-or-later.** For some organisations this is a hard adoption gate. Plenty of
corporate policies restrict or forbid AGPL, and its network-copyleft clause is unusual. For how
SysCert actually runs, a local CLI on a timer rather than a network service you've modified and
exposed, that clause is largely inert, and running the unmodified binary puts no obligation on you.
But if your policy bans AGPL outright, clear that before anything else here matters.

**Output lock-in is effectively nil.** SysCert writes certbot-compatible artefacts
(`cert`/`privkey`/`chain`/`fullchain`.pem, plus an optional bundle) and speaks standard ACME. Leave,
and your consumers keep reading the same files while you re-point them at certbot, cert-manager, or
Vault's own agent without touching the apps that consume the certs. Low switching cost is a risk
control in its own right.

## 4. Production / operational risk

The risk of running the service day to day.

**The primary failure mode is silent non-renewal.** What actually hurts you is a certificate quietly
failing to renew: a transient CA or DNS outage that never recovers, a rotated DNS credential, a CA
policy change, going unnoticed until the cert expires and TLS breaks on that host. That's the one to
watch. The blast radius is contained, though. Failures are per-host with no shared control plane, and
the outcome is an expired certificate, an availability event on one host rather than a breach or a key
compromise.

**Observability, and its gap.** SysCert logs to the journal, exits non-zero on failure (so
`systemctl status` or a failed-unit alert catches a bad run), and ships a read-only `syscert status`
that reports the stored certificate's expiry and renewal dates. What it won't do is alert you. There's
no built-in metrics endpoint and no "cert expiring / renewal failing" notification. Closing that gap
is your job: wire the unit's failure state and the certificate's expiry into whatever you already run,
a systemd `OnFailure=`, a node-exporter textfile, an expiry probe. Treat it as a required integration,
not an optional nicety.

**Backup and recovery.** SysCert doesn't back itself up. The state that matters is `/var/lib/syscert`
(the ACME account key, current keys and certs) and `/etc/syscert` (config and secrets). Recovery is
mostly reproducible: lose a host, and a reinstall plus a run re-issues fresh certificates against your
CA. Keeping the **ACME account key** and the **secrets** file saves you re-registering and
re-supplying credentials. The [Recover procedure](/docs/procedures/recover/) has the steps; running
and testing a restore is on you.

**Availability and DR.** No shared state, nothing to cluster. Each host is independent and stateless
beyond its own store. DR is "redeploy the binary and config, then run," which the
[offline bundle](/docs/advanced-install/offline/) makes possible even with no internet. There's no
high-availability story because the service doesn't need one; a missed run is retried on the next
timer tick.

**Privilege and isolation.** Runs as a dedicated non-root `syscert` user with a single capability
(`CAP_CHOWN`) inside a hardened systemd sandbox. Low operational risk here, assessed in detail in the
[Security assessment §5](/docs/compliance/security/).

## 5. Supply-chain risk

**Release integrity.** Release binaries ship with `sha256sums.txt` and a SLSA build-provenance
attestation (GitHub OIDC), built reproducibly (`-trimpath`, pinned `go.sum`, `CGO_ENABLED=0`). You can
check origin and integrity yourself with `sha256sum --check` and `gh attestation verify` before
anything runs. See [Security assessment R-07 / F-05](/docs/compliance/security/).

**The install channel is the weakest link, so don't trust it.** The convenience one-liner
(`curl … | sudo sh`) trusts `syscert.tfindley.dev`, GitHub, and TLS in the moment. For anything you
care about, skip it. Pin `SYSCERT_VERSION` and verify the checksum and provenance yourself, or better,
build an [offline bundle](/docs/advanced-install/offline/) once, verify it, and install every host
from your **own** internal mirror. The installer already supports that: it takes a local binary path
and needs no network.

**Or drop the published-binary trust altogether.** Two direct dependencies and a static build mean you
can `go build` SysCert from pinned source and distribute your own binary. Then the only supply-chain
trust left is the Go toolchain and the dependency tree, both of which you audit on your own terms.

**Dependency tree.** The large transitive tree comes almost entirely from `lego` vendoring every
DNS-provider SDK, and only the provider you configure ever runs. Pinned `go.sum` plus the mandatory
`govulncheck` gate (0 reachable vulnerabilities at the last assessment) keep this at Low residual. See
[Security assessment R-01](/docs/compliance/security/).

## 6. Supplier independence — your exit strategy

The honest answer to "what if the maintainer vanishes?" isn't "trust that they won't." It's a plan for
carrying on without them. A cautious adopter should do some of this before going live, not after
something breaks:

1. **Mirror the source.** Vendor the repository at the tag you deploy. AGPL guarantees you can always
   get the corresponding source; make sure *you* actually have it, in your own git, not just a link to
   GitHub.
2. **Self-host the binaries.** Build an offline bundle (or `go build` from source), verify it once,
   and serve it from your internal artefact store. Install every host from there. Now your deployment
   path has no external dependency.
3. **Be ready to fork.** The codebase is small and the dependencies are few. If upstream stalls on a
   fix you need, forking and patching is an afternoon, not a project.
4. **Keep an exit route.** Because the outputs are certbot-compatible and the protocol is standard
   ACME, migrating away is a config change on the consumers, not a re-architecture. Know the route's
   there; you may never use it.

Do that, and "single-maintainer, pre-1.0" stops being a blocker. It becomes a manageable, priced-in
risk.

## 7. FitSD Service Acceptance Criteria mapping

Mapped against the nine named criteria of the [FitSD](https://fitsd.tfindley.dev) Service Acceptance
Criteria (the Definition of Done in FSD-PRO §7, required by FSD-SD-5, evidenced via FSD-FRM-03).
FitSD's own supplier/third-party capability (FSD-SC) is a known gap in that framework; this document,
plus the Gate 1 licensing/upgrade-path check, is where SysCert's supplier due-diligence sits.

Verdicts: **Met** means SysCert provides it and it's evidenced. **Partial** means it's provided but
with a gap you have to close. **Operator** means it's inherently your responsibility, with SysCert
giving you what you need to satisfy it.

| SAC criterion (FSD-PRO §7) | What SysCert gives you | Gap / what you must own | Verdict |
|---|---|---|---|
| **Documentation** | Architecture + trust model ([Security assessment](/docs/compliance/security/), [Tech stack](/docs/compliance/tech-stack/)), a full runbook set ([Procedures](/docs/procedures/)), a [Recover procedure](/docs/procedures/recover/), and user docs | Record the links in your own service record | **Met** |
| **Backup (tested)** | Documented recovery; reproducible re-issuance; clearly-scoped state (`/var/lib/syscert`, `/etc/syscert`) | SysCert doesn't back itself up — you define scope/retention and **perform a test restore** | **Operator** |
| **Security** | Published, tool-backed [Security assessment](/docs/compliance/security/); `gosec`/`govulncheck` gates; hardened non-root unit; a patch path via releases | Track advisories; keep the version current | **Met** |
| **Access** | Dedicated non-root user, least privilege, `CAP_CHOWN` only, explicit file modes, store-ownership preflight | Host-level joiners/movers/leavers and who may edit `/etc/syscert` are your IAM | **Met** (service) |
| **Availability** | Stateless per-host; missed runs retried next tick; DR = redeploy (works offline) | Set your own expectation; there is no HA because none is needed | **Partial** |
| **Monitoring & alerting** | Journal logs, non-zero exits, `syscert status` with expiry/renewal dates | **No built-in alerting** — you must wire cert-expiry and unit-failure alerts end to end | **Partial** |
| **Incident profile** | Clear signals: unit failure, imminent expiry, renewal errors | You define what counts as an incident for *this* service and register it with your incident process | **Operator** |
| **Supportability / handover** | Runbooks/SOPs, ADR log, changelog | Operating knowledge is well-captured; **continuity of the *supplier* is the open risk** — mitigate per §6 (fork-readiness) | **Partial** |
| **Cost / licensing** | Free; run-cost is a periodic timer (negligible) | Clear the **AGPL-3.0** licence against your policy (§3) | **Met** (with licence check) |

The split is consistent. SysCert cleanly meets the criteria about the product itself (security,
access, documentation, licensing) and hands you the operational ones (backup, monitoring, incident
definition) with the signals and docs to satisfy them. Two amber items deserve real attention:
monitoring and alerting, a genuine feature gap you have to close, and supplier continuity, which
fork-readiness mitigates rather than trust.

## 8. Risk register

Same scale as the [Security assessment](/docs/compliance/security/): Likelihood (L) / Impact (I) /
Residual as **L**ow / **M**edium / **H**igh; inherent = pre-mitigation. IDs are `RA-nn` to keep them
distinct from the security register's `R-nn`.

| ID | Risk | Category | L | I | Inherent | Key controls / mitigations | Residual | SAC |
|---|---|---|---|---|---|---|---|---|
| RA-01 | Maintainer abandons the project; no future fixes/releases | Supplier viability | M | M | Medium | AGPL source; 2 direct deps; small codebase; **no runtime dependency on the maintainer** (running hosts keep working); fork-ready (§6) | **Medium** | Supportability |
| RA-02 | Silent non-renewal → certificate expires → TLS outage on a host | Operational | M | H | High | Per-host blast radius; non-zero exit + failed-unit signal; `status` expiry; retried next tick — **requires** operator alerting (RA-03) | **Medium** | Monitoring; Availability |
| RA-03 | Expiry/failure goes unnoticed (no built-in alerting) | Observability | M | H | High | Journal + exit codes + `status` provide the signals; operator must wire expiry + unit-failure alerts | **Medium** | Monitoring & alerting |
| RA-04 | Compromised or unreachable install channel (`curl \| sh`) | Supply chain | L | H | Medium | `sha256sums.txt` + SLSA provenance; pin `SYSCERT_VERSION`; offline bundle from an internal mirror; build from source | **Low** | Security |
| RA-05 | AGPL-3.0 conflicts with organisational policy | Legal / licensing | L | M | Medium | Local-CLI use makes the network clause inert; unmodified-binary use imposes no obligation; check policy up front | **Low** | Cost / licensing |
| RA-06 | Store/secrets loss with no tested restore | Continuity | L | M | Medium | Reproducible re-issuance; documented [Recover](/docs/procedures/recover/); operator backs up account key + secrets and tests restore | **Low** | Backup (tested) |
| RA-07 | Pre-1.0 breaking change on upgrade | Change management | M | L | Medium | SemVer + Conventional Commits + changelog + ADR log; pin versions; read the changelog before upgrading | **Low** | Supportability |
| RA-08 | Vulnerable/malicious transitive dependency | Supply chain | M | M | Medium | Pinned `go.sum`; mandatory `govulncheck` gate; only the configured provider runs — see [R-01](/docs/compliance/security/) | **Low** | Security |
| RA-09 | Operating knowledge concentrated in one operator | Continuity | L | M | Medium | Runbooks/SOPs, ADRs, changelog capture the knowledge — cross-train off the docs | **Low** | Supportability / handover |

**Residual profile:** three rows sit at Medium: supplier viability (RA-01) and the expiry/alerting
pair (RA-02, RA-03). None is a code defect. Each closes the same way, by the adopter owning something,
an exit plan and monitoring. Everything else reduces to Low.

## 9. Recommendations for adopters

1. **Close the alerting gap before go-live.** Alert on `syscert.service` failure and on certificate
   expiry approaching. This is the single highest-value mitigation (RA-02, RA-03).
2. **Own your supply chain.** Verify provenance and checksums, then install from an internal mirror
   via the [offline bundle](/docs/advanced-install/offline/), or build from source (RA-04).
3. **Have an exit plan on day one.** Mirror the source, self-host binaries, be fork-ready (§6, RA-01).
4. **Back up and test a restore.** Preserve the ACME account key and `/etc/syscert/secrets`; rehearse
   [Recover](/docs/procedures/recover/) (RA-06).
5. **Clear the AGPL licence** against policy before you invest further (RA-05).
6. **Pin and track versions.** Read the changelog before each upgrade (RA-07).
7. **Define the incident profile.** Decide what "renewal failing" and "expiry imminent" mean for your
   service and register them with your incident process.

## 10. Conclusion

SysCert is low technical risk and acceptable to adopt, as long as you go in clear about what you're
accepting. The code is memory-safe, runs least-privilege, and comes with a clean assessment. The
outputs are standard. And the point that carries the most weight: running it creates no dependency on
the maintainer at all.

The risks that remain aren't about the software being bad. They're about it being *small*. One
maintainer, no built-in alerting, plus the ordinary operator duties of backup and monitoring. Every
one of those is manageable, and manageable precisely because this is open source you can fork, build,
and run entirely on your own infrastructure. Treat it as software you could take over tomorrow and
it's a sound choice. Treat it as a vendor relationship you're trusting to last, and you've misread the
risk.

---

### Appendix — how to verify the claims in this document

Don't take the assessment on faith; that would defeat the point of it.

```sh
# Two direct dependencies, nothing hidden
grep -A3 '^require' go.mod

# No phone-home: the only external hosts are the LE directory defaults + doc URLs in help text
grep -rnoE 'https?://[^"]+' cmd/ internal/ | grep -v _test.go

# Release integrity, independently
sha256sum --check sha256sums.txt
gh attestation verify syscert-linux-amd64 --repo tfindley/syscert

# Build it yourself and compare
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o syscert ./cmd/syscert
```

This assessment reflects SysCert around v0.4.0 and the codebase at the time of writing. It's a
maintainer-published document, and the whole design is that you can re-derive every claim from the
source instead of trusting the author. Re-run the checks above, and disagree wherever the evidence
tells you to.
