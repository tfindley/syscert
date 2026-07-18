---
title: Embedded pattern
navLabel: Embedded
description: Embed syscert and nginx in a single container image — with mandatory caveats and a restart-on-failure mitigation to avoid silently serving a stale certificate.
order: 3
eyebrow: "// docs · containerisation · embedded"
lede: One container, two processes. It works, but it has a real downside. Read the trade-off and the mitigation before you reach for it.
---

> **Read the trade-off below before you use this pattern.** If a single-container constraint isn't
> forced on you, use the [sidecar pattern](/docs/containerisation/sidecar/) instead.

The embedded pattern runs `syscert --interval` and the application (nginx, say) in a **single
container**. It's here because it works and is sometimes forced on you (a constrained PaaS,
image-size budgets, a demo), but it comes with a real trade-off that you have to mitigate on purpose.

## The trade-off

Two jobs in one container gives you an ambiguous failure mode:

- If **syscert** fails (a persistent ACME error, say), nginx keeps running and serves the existing,
  possibly expiring certificate **silently**. Nothing tells you the renewal broke.
- If **nginx** crashes, syscert keeps renewing, but now nothing is serving the certificate.

Neither failure makes the container exit, so Docker's restart policy never fires. Both stay
silent unless you're watching the log output.

## Required mitigation: restart on either exit

The `entrypoint.sh` in the example starts both processes and **exits the script the moment either one
exits**. Docker's `restart: unless-stopped` then restarts the whole container, which at least
surfaces the failure in the restart count and the container logs.

```sh
#!/bin/sh
set -e

syscert dry-run --config-only --config /etc/syscert/syscert.toml

nginx -g "daemon off;" &
NGINX_PID=$!

syscert --interval 12h --config /etc/syscert/syscert.toml &
SYSCERT_PID=$!

# Reload nginx on cert change (Linux-only; remove if inotifywait is not available).
if command -v inotifywait >/dev/null 2>&1; then
  (while inotifywait -e close_write /var/lib/syscert 2>/dev/null; do nginx -s reload; done) &
fi

wait -n "$NGINX_PID" "$SYSCERT_PID" 2>/dev/null || wait "$NGINX_PID" "$SYSCERT_PID"
```

For something sturdier, swap this script for a real process supervisor:

- **[s6-overlay](https://github.com/just-containers/s6-overlay)** — the most widely used in Alpine
  images; per-service restart policies, dependency ordering, clean init.
- **runit** — lightweight, good for BusyBox-based images.
- **tini** (`--init` in Docker) — for reaping zombie processes, but not a full supervisor.

A supervisor gives you per-process restart policies (restart syscert on failure without touching
nginx, and the other way round) plus proper PID 1 signal handling.

## Dockerfile

```dockerfile
FROM nginx:alpine

ARG SYSCERT_VERSION=v0.3.1
ARG TARGETARCH=amd64

RUN apk add --no-cache curl && \
    curl -fsSL \
      "https://github.com/tfindley/syscert/releases/download/${SYSCERT_VERSION}/syscert-linux-${TARGETARCH}" \
      -o /usr/local/bin/syscert && \
    chmod 0755 /usr/local/bin/syscert && \
    apk del curl

RUN adduser -D -u 1001 -G nginx syscert && \
    mkdir -p /var/lib/syscert /etc/syscert && \
    chown syscert:nginx /var/lib/syscert && \
    chmod 0750 /var/lib/syscert

COPY syscert.toml /etc/syscert/syscert.toml
COPY entrypoint.sh /entrypoint.sh
RUN chmod 0755 /entrypoint.sh

EXPOSE 80 443
ENTRYPOINT ["/entrypoint.sh"]
```

Full annotated files:
[`examples/container/embedded/Dockerfile`](https://github.com/tfindley/syscert/blob/main/examples/container/embedded/Dockerfile)
and
[`examples/container/embedded/entrypoint.sh`](https://github.com/tfindley/syscert/blob/main/examples/container/embedded/entrypoint.sh)

## Challenge

Use `challenge = "dns-01"` in `syscert.toml`. No inbound ports, and nothing to fight nginx over
`:80`/`:443`. See [Containerisation overview → Challenge selection](/docs/containerisation/#challenge-selection----dns-01-is-the-right-choice-in-containers).

## Secrets

Pass DNS-provider credentials in at runtime with `--env-file` or Docker secrets. Never bake them
into the image.

```sh
docker run --env-file .env myapp:latest
```

## Volume ownership

The store at `/var/lib/syscert` has to be owned by the uid syscert runs as. In the Dockerfile
above, `adduser` creates uid `1001`, and `chown syscert:nginx` sets ownership so nginx (gid `101`)
can read the distributed files.

With a named volume (worth it for persistence), fix its owner first:

```sh
docker run --rm -v <project>_certs:/var/lib/syscert myapp:latest chown syscert:nginx /var/lib/syscert
```

## When to use this pattern

Use the embedded pattern when:
- A **single-container constraint** is hard (PaaS, image registry policy, demo packaging).
- You need to minimize the number of running containers (cost, orchestrator limits).

Use the [sidecar](/docs/containerisation/sidecar/) or [scheduled](/docs/containerisation/scheduled/)
pattern when:
- You want clear failure isolation between cert renewal and the application.
- You need to update syscert or nginx independently.
- You're operating at any scale past a single instance.

---

Next: [Sidecar pattern](/docs/containerisation/sidecar/) ·
[Scheduled pattern](/docs/containerisation/scheduled/) ·
[Containerisation overview](/docs/containerisation/)
