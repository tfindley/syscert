---
title: Sidecar pattern
navLabel: Sidecar
description: Run syscert as a long-running sidecar alongside nginx — it renews on an interval and distributes into a shared cert volume the app reads.
order: 1
eyebrow: "// docs · containerisation · sidecar"
lede: Two containers, one shared cert volume. syscert renews on --interval 12h; nginx reads the volume read-only. The recommended pattern for long-lived containers.
---

The sidecar pattern is **recommended** for services that run continuously. `syscert --interval 12h`
renews in a loop; nginx (or any other service) reads certificates from a shared volume. Two clean
containers, each with one responsibility.

## compose.sidecar.yml

```yaml
services:
  syscert:
    image: alpine:latest          # replace with your own image containing syscert
    user: "1000:1000"             # run as a non-root uid; must own the cert volume
    command:
      - syscert
      - --interval
      - 12h
      - --config
      - /etc/syscert/syscert.toml
      - --env-file
      - /run/secrets/syscert_secrets
    volumes:
      - ./syscert.toml:/etc/syscert/syscert.toml:ro
      - certs:/var/lib/syscert
    env_file:
      - .env
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - certs:/etc/nginx/tls:ro
    depends_on:
      - syscert
    restart: unless-stopped

volumes:
  certs:
    driver: local
```

Full annotated file:
[`examples/container/compose.sidecar.yml`](https://github.com/tfindley/syscert/blob/main/examples/container/compose.sidecar.yml)

## How `--interval` works

`syscert --interval 12h` runs the full ensure cycle (issue if missing, renew if due, distribute),
then sleeps 12 hours, then repeats. A failed cycle (transient ACME or DNS error) is logged and the
loop continues — it does not crash the container. A bad configuration exits non-zero **before** the
loop starts, so misconfiguration is caught immediately.

`SIGTERM` (the signal Docker sends on `docker stop`) finishes the current cycle and exits cleanly
within Docker's 30-second grace period.

## Reloading nginx

SysCert never runs reload commands (see [Reloading services](/docs/reloading/)). nginx does not
auto-reload on cert change — wire a reload separately:

- **Periodic cron inside nginx** (simplest for most setups):

  ```sh
  # In your nginx Dockerfile or init script:
  echo "0 */12 * * * nginx -s reload" | crontab -
  ```

- **The optional `reload-helper.sh`** — an `inotifywait` loop that reloads on store writes. See
  [`examples/container/reload-helper.sh`](https://github.com/tfindley/syscert/blob/main/examples/container/reload-helper.sh).
  Linux-only; requires `inotify-tools`.

- **A separate compose service** that runs `docker exec nginx nginx -s reload` on a timer.

Pick the one that fits your setup. The cron approach is the most portable.

## Volume ownership

The `certs` volume must be owned by the uid syscert runs as (`1000` above). Fix it before first run:

```sh
docker run --rm -v <project>_certs:/vol alpine chown 1000:1000 /vol
```

A wrong owner causes `permission denied` on the store write — syscert logs the error and retries on
the next interval, but until it succeeds nginx has no certificate.

## Secrets

DNS-provider credentials go in the `.env` file (mode `0600`) or Docker secrets — never in the image
or the compose file. The `env_file` directive loads them into the container's environment.

```sh
# .env (0600)
CLOUDFLARE_DNS_API_TOKEN=your-token-here
```

Look up your provider's variable names at the [lego DNS provider
list](https://go-acme.github.io/lego/dns/).

## Config (syscert.toml)

Use `challenge = "dns-01"` — it needs no inbound ports and cannot conflict with nginx's `:80`/`:443`.
See [`examples/container/syscert.toml`](https://github.com/tfindley/syscert/blob/main/examples/container/syscert.toml)
for a ready-to-edit starter, or the [Configuration reference](/docs/configuration/).

---

Next: [Scheduled pattern](/docs/containerisation/scheduled/) ·
[Embedded pattern](/docs/containerisation/embedded/) ·
[Containerisation overview](/docs/containerisation/)
