---
title: Sidecar pattern
navLabel: Sidecar
description: Run syscert as a long-running sidecar alongside nginx — it renews on an interval and distributes into a shared cert volume the app reads.
order: 1
eyebrow: "// docs · containerisation · sidecar"
lede: Two containers, one shared cert volume. syscert renews on --interval 12h; nginx reads the volume read-only. The recommended pattern for long-lived containers.
---

For anything that runs continuously, the sidecar pattern is **recommended**. `syscert --interval 12h`
renews in a loop, and nginx (or any other service) reads certificates from a shared volume. Two clean
containers, each with a single job.

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
sleeps 12 hours, then goes again. A failed cycle from a transient ACME or DNS error gets logged and
the loop carries on; it won't crash the container. A bad configuration is different: it exits
non-zero **before** the loop even starts, so you catch a misconfiguration right away.

`SIGTERM` (what Docker sends on `docker stop`) finishes the current cycle and exits cleanly inside
Docker's 30-second grace period.

## Reloading nginx

SysCert never runs reload commands (see [Reloading services](/docs/reloading/)). nginx won't reload
itself when the cert changes, so wire up a reload on your own:

- **Periodic cron inside nginx** (simplest for most setups):

  ```sh
  # In your nginx Dockerfile or init script:
  echo "0 */12 * * * nginx -s reload" | crontab -
  ```

- **The optional `reload-helper.sh`** — an `inotifywait` loop that reloads on store writes. See
  [`examples/container/reload-helper.sh`](https://github.com/tfindley/syscert/blob/main/examples/container/reload-helper.sh).
  Linux-only; requires `inotify-tools`.

- **A separate compose service** that runs `docker exec nginx nginx -s reload` on a timer.

Pick whichever fits your setup. Cron travels the best.

## Volume ownership

The `certs` volume has to be owned by the uid syscert runs as (`1000` above). Fix it before the first run:

```sh
docker run --rm -v <project>_certs:/vol alpine chown 1000:1000 /vol
```

Get the owner wrong and the store write fails with `permission denied`. syscert logs it and retries
on the next interval, but until that succeeds nginx has no certificate.

## Secrets

DNS-provider credentials go in the `.env` file (mode `0600`) or Docker secrets. Never in the image
or the compose file. The `env_file` directive loads them into the container's environment.

```sh
# .env (0600)
CLOUDFLARE_DNS_API_TOKEN=your-token-here
```

Look up your provider's variable names at the [lego DNS provider
list](https://go-acme.github.io/lego/dns/).

## Config (syscert.toml)

Use `challenge = "dns-01"`. It needs no inbound ports and can't collide with nginx's `:80`/`:443`.
See [`examples/container/syscert.toml`](https://github.com/tfindley/syscert/blob/main/examples/container/syscert.toml)
for a ready-to-edit starter, or the [Configuration reference](/docs/configuration/).

---

Next: [Scheduled pattern](/docs/containerisation/scheduled/) ·
[Embedded pattern](/docs/containerisation/embedded/) ·
[Containerisation overview](/docs/containerisation/)
