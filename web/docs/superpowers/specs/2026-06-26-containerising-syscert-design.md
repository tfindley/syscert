# Design — Containerising syscert

**Date:** 2026-06-26
**Status:** Approved (brainstorm) — ready for implementation planning
**Scope:** One cohesive feature ("run syscert in containers"), four components built in sequence.

## Motivation

syscert is host/systemd-oriented today: a non-root `syscert` user runs the one-shot binary under a
systemd timer, writing to `/var/lib/syscert` and distributing to consumer paths, **running no reload
commands** (consumers watch + reload themselves — e.g. a `systemd.path`). A new use case has emerged:
embedding syscert into a Docker container (nginx/apache) so **the container is responsible for its own
TLS certificate**, with persistent cert storage on the host.

syscert is already most of the way there — the timer is just *one* scheduler around a static one-shot
binary. Containerising needs: a non-systemd scheduler, a persistent volume for the store (the **ACME
account key** above all), and a reload mechanism — all solvable without compromising syscert's
command-free security model.

## Non-goals

- **No reload hooks in syscert.** It keeps its command-free invariant; reload stays a container-layer
  concern (a watcher, a periodic reload, or an orchestrator action). Same contract as the host's
  `systemd.path`, different watcher.
- **No Traefik integration.** The "behind Traefik" case is *TLS passthrough* — pure Traefik config;
  syscert never touches Traefik (see §4).
- **No orchestrator operator/controller.** We ship example manifests, not a k8s operator.
- **The host/systemd path is unchanged.** This is additive.

## Cross-cutting decisions

- **Default challenge: `dns-01`** — no inbound ports, works behind NAT/proxies/passthrough. http-01 /
  tls-alpn-01 are documented with their reachability caveats (the CA must reach the container on
  :80/:443 — awkward when a proxy fronts those ports).
- **Runs non-root**; key-bearing files stay `0600` (syscert already enforces this). The store volume
  must be writable by the image's non-root uid (documented).
- **Secrets** via environment / Docker secrets / `--env-file` — never baked into the image or the TOML.
- All existing syscert invariants hold (fresh keypair per renewal, certbot-compatible outputs, etc.).

---

## Component 1 — `--interval` mode (the only binary change)

A native loop so the binary can schedule itself where there's no systemd (containers **and**
appliances).

- New flag **`--interval <duration>`** (+ env **`SYSCERT_INTERVAL`**) on the bare/`ensure` path.
  Behaviour: run `ensure` immediately, then `sleep(interval)` and repeat, indefinitely. **No flag =
  today's one-shot** (run once, exit) — fully backward-compatible.
- **Signals:** `SIGTERM`/`SIGINT` → clean exit (0) at the next safe point (don't interrupt an
  in-flight issuance/write; interrupt the sleep). Containers send `SIGTERM` on stop.
- **Per-cycle errors:** a failed cycle (transient ACME/DNS/network) **logs and continues** to the next
  cycle — a sidecar must survive blips. Fatal config/startup errors still exit non-zero *before* the
  loop starts (validation runs once up front).
- **Duration:** Go `time.ParseDuration`; reject implausibly small intervals (floor, e.g. `< 1m`) to
  avoid hammering the CA.
- **Tests (TDD):** injected clock/sleeper (no real sleeping) — N cycles run; one-shot default
  unchanged; signal → clean exit; a failing cycle continues; sub-floor interval rejected.
- The cron/appliance docs gain a note that `--interval` is an alternative to crond.

## Component 2 — published image `ghcr.io/tfindley/syscert`

- **Base: `gcr.io/distroless/static:nonroot`** — just the CGO-free static binary, CA roots included,
  non-root uid. No shell/package manager → minimal attack surface (same rationale as the alpine-slim
  web image). Multi-stage: a Go builder stage compiles `-trimpath -ldflags "-s -w"`, final stage is
  distroless.
- `ENTRYPOINT ["/usr/local/bin/syscert"]`; default invocation is the one-shot. Sidecar runs add
  `--interval 12h`.
- **Paths:** config mounted at `/etc/syscert/syscert.toml`; store volume at `/var/lib/syscert`
  (persists the ACME account key + certs). Document the non-root uid (distroless `nonroot` = 65532)
  so operators set volume ownership correctly.
- **CI:** built/scanned/published **on release** so the image version tracks the binary. Multi-arch
  (amd64/arm64) buildx; **Grype** image scan + **govulncheck**; GHCR push with `vX.Y.Z` + `latest` +
  `sha-` tags and a **SLSA provenance attestation** — mirrors the binary release pipeline.
- No shell ⇒ no shell `HEALTHCHECK`; rely on orchestrator + logs. (Optional future: a binary-based
  health subcommand — deferred.)

## Component 3 — deployment patterns + reference examples (`examples/container/`)

All default to `dns-01`, mount the store volume, and take secrets via env/Docker secrets. The reload
is per-pattern, always at the container layer.

- **Embedded** (`examples/container/embedded/`) — `Dockerfile` (`FROM nginx:alpine`) that adds the
  syscert binary, plus an entrypoint that runs `syscert --interval` in the background and a small
  **inotify watcher** (`reload-helper.sh`) on the cert path → `nginx -s reload`, then `exec nginx`.
  Same PID namespace ⇒ reload is direct. The container owns its cert end-to-end. Includes a compose
  file mounting the store volume + secrets.
- **Sidecar** (`examples/container/compose.sidecar.yml`) — the published image (`--interval 12h`)
  distributes into a cert volume shared (read-only) with an `nginx` service. Reload via either the
  shipped **`reload-helper.sh`** (inotify) running in the nginx container, or a **periodic graceful
  reload** loop — documented as the two no-privilege options. No docker-socket, no shared PID
  namespace required.
- **Scheduled job** (`examples/container/compose.scheduled.yml` + a **k8s `CronJob`** snippet) —
  `docker run --rm syscert` (one-shot, no `--interval`) on an external timer, writing into a volume
  the app reads. Reload via app restart or a periodic reload.
- **`reload-helper.sh`** — a ~20-line `inotifywait` loop, parameterised by watch-path + reload-command;
  shared by the embedded and sidecar patterns.

## Component 4 — behind a TLS-passthrough proxy (niche, doc-only)

Scenario: the container is fronted by Traefik, but it should **own its TLS end-to-end** rather than let
Traefik terminate. This is **TLS passthrough** — a Traefik **TCP router** with an SNI `HostSNI(...)`
rule and **`tls.passthrough=true`**, no certresolver/termination for that router. The container (its
cert managed by syscert via any pattern above) terminates TLS itself; Traefik just routes the encrypted
stream through. **syscert is unaware of Traefik.** This forces `dns-01` (the CA can't reach the
container on :443 when Traefik fronts it). Documented as a short subsection with the Traefik
label/dynamic-config snippet; flagged niche.

## Component 5 — docs: a top-level **Containerisation** section

The nav nests one level only (`DocsLayout.astro` parents a page by its first path segment), so this is
a **new top-level section** (not nested under Advanced install), giving each option its own page:

- `docs/containerisation.md` — overview: when to containerise, the three patterns at a glance, the
  dns-01 / persistence / secrets fundamentals, the published image, and the TLS-passthrough subsection.
- `docs/containerisation/embedded.md`, `docs/containerisation/sidecar.md`,
  `docs/containerisation/scheduled.md` — one page per option (order 1/2/3), each with its worked
  example from `examples/container/`, the reload approach, and persistence/secrets notes.
- Menu placement near the other install/deployment topics (exact `order` set during implementation,
  minimising renumbering). Vendored as usual; **no version bump** for the docs.
- Cross-link from the relevant Procedures SOPs (e.g. Install & deploy) and from the cron page (which
  gains the `--interval` note).

---

## Build sequence

1. **`--interval` mode** (binary, TDD) — foundational; everything else uses it.
2. **Published image + CI** (distroless Dockerfile, multi-arch buildx, scan, GHCR publish on release).
3. **Reference examples** (`examples/container/`: embedded Dockerfile, the two compose files, the k8s
   CronJob snippet, `reload-helper.sh`).
4. **Docs** (the Containerisation section + sub-pages; passthrough subsection; cron `--interval` note).

## Testing

- **Unit (Component 1):** the interval loop via an injected clock — multi-cycle, one-shot-default,
  signal-clean-exit, error-continue, sub-floor rejection.
- **Image (Component 2):** build multi-arch; run one-shot and `--interval`; confirm persistence of the
  account key + cert across container restarts (named volume); a real `dns-01` issuance.
- **Examples (Component 3):** compose smoke test per pattern; confirm nginx serves the issued cert and
  reloads on renewal (force a renewal, watch the reload).
- Fold the container cases into the manual QA plan (`docs/internal/qa.md`) as a new section.
