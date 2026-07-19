---
title: Containerisation
navLabel: Containerisation
description: Run syscert in a container — sidecar, scheduled one-shot, or embedded — so the container owns its own TLS certificate end-to-end via dns-01.
order: 7.5
eyebrow: "// docs · containerisation"
lede: A container that terminates TLS directly can own its own certificate renewal. syscert fits that niche — it's a static binary that issues, renews, and distributes, then gets out of the way.
---

SysCert's default model is a host systemd timer that fires a one-shot binary. That same binary runs just as well in a container, either as a long-running sidecar (`--interval`) or as a one-shot job on an external schedule. Either way, the container owns its TLS end-to-end.

## Challenge selection — dns-01 is the right choice in containers

> **dns-01 is the default and the correct challenge for containers.**
>
> `http-01` and `tls-alpn-01` bind `:80` and `:443` directly from inside syscert; they stand up
> syscert's **own** servers on those ports. If nginx, Apache, or Traefik is already listening there
> (as it usually is in a container serving HTTPS), syscert fails right away with
> **`address already in use`**. There's no webroot mode and no port-sharing.
>
> **If you need http-01 behind a running web server, syscert isn't the tool for it. Use
> `certbot --webroot` or a purpose-built ACME client with webroot support.**
>
> `dns-01` needs no inbound ports and no coordination with other processes. For a container serving
> traffic on `:80`/`:443`, it's the only practical challenge.

## The three patterns

| Pattern | When to use | Complexity |
|---|---|---|
| [**Sidecar**](/docs/containerisation/sidecar/) | Long-lived container that self-renews | Low — two services, one shared volume |
| [**Scheduled**](/docs/containerisation/scheduled/) | Simplest; external timer; k8s-native | Lowest — one-shot, no loop |
| [**Embedded**](/docs/containerisation/embedded/) | Single-container constraint | Higher — two processes, needs mitigation |

Start with the sidecar or scheduled pattern. The embedded pattern has caveats that need mitigation, so read its page before you reach for it.

## Persistence: the cert volume

SysCert writes the certificate store to `/var/lib/syscert` (the `[store] path`). In a container that path has to be a **named volume** or a host mount, never the container's ephemeral layer. Skip persistence and every container start kicks off a fresh ACME order, which burns through CA rate limits fast.

## Volume ownership — the silent EACCES gotcha

SysCert runs as a non-root user. The cert volume **must be owned by the same uid** that syscert runs as. Otherwise syscert fails with a permission error the first time it tries to write the store.

```sh
# Fix ownership before first use (replace 1000 with your uid):
docker run --rm -v <project>_certs:/vol alpine chown 1000:1000 /vol
```

In Kubernetes the `securityContext.fsGroup` field on the Pod spec handles this for you; set it to match `runAsGroup`.

What the wrong owner looks like: syscert starts, passes config validation, then fails with `permission denied` (or `EACCES`) trying to write the store. The volume stays empty, and the app has nothing to serve.

## Secrets

DNS-provider credentials (e.g. `CLOUDFLARE_DNS_API_TOKEN`) have to reach the container at runtime. Never bake them into the image. Two ways:

- Docker Compose: `env_file: [.env]` (file mode `0600`) or Docker secrets via `/run/secrets/`.
- Kubernetes: a `Secret` projected as environment variables or a volume.

The `--env-file` flag loads an env file when there's no native way to inject environment variables (say, on a NAS or in some CI setups).

## TLS passthrough (Traefik / ingress)

When a reverse proxy sits in front of the container, set it to **TLS passthrough** (TCP/SNI proxy) rather than TLS termination. The container terminates TLS itself; the proxy just forwards raw TCP. SysCert never knows the proxy is there.

Traefik example (EntryPoints + TCPRouter with `passthrough: true`):

```yaml
# traefik dynamic config
tcp:
  routers:
    myapp-tls:
      rule: HostSNI(`app.example.com`)
      service: myapp
      tls:
        passthrough: true
  services:
    myapp:
      loadBalancer:
        servers:
          - address: "myapp:443"
```

This is where syscert shines. The container owns the cert, and the proxy stays out of the TLS path.

## Ephemeral containers

Don't lean on `--interval` mode in containers that restart often, like serverless, spot, or autoscaled replicas. Every restart runs a new cert-check cycle. That part's fine, but if the store volume isn't shared across replicas, **each replica issues its own certificate** and burns through CA rate limits.

For ephemeral or scaled containers, reach for the **scheduled** pattern with a shared PVC, or a central cert store that the replicas mount read-only.

## Pattern pages

- [Sidecar](/docs/containerisation/sidecar/) — `compose.sidecar.yml`
- [Scheduled](/docs/containerisation/scheduled/) — `compose.scheduled.yml` + `k8s-cronjob.yaml`
- [Embedded](/docs/containerisation/embedded/) — `embedded/Dockerfile` + `entrypoint.sh`

Example files live in [`examples/container/`](https://github.com/tfindley/syscert/tree/main/examples/container/).

---

Next: [Reloading services](/docs/reloading/) · [Configuration](/docs/configuration/) · [Troubleshooting](/docs/troubleshooting/)
