---
title: Scheduled pattern
navLabel: Scheduled
description: Run syscert as a one-shot job on an external timer — the simplest container pattern, and the Kubernetes-native path via CronJob.
order: 2
eyebrow: "// docs · containerisation · scheduled"
lede: The simplest pattern. syscert runs once, exits, and an external timer brings it back. Mirrors the systemd-timer model and maps directly to a Kubernetes CronJob.
---

The scheduled pattern is **recommended** for most setups. `syscert` runs without `--interval`,
performs one ensure cycle (issue if missing, renew if due, distribute), and exits. An external
timer — cron, a systemd timer on the host, or a Kubernetes CronJob — re-runs it on a schedule.

This mirrors the host systemd-timer model: a one-shot binary on a schedule, easy to inspect, trivially
restartable, and with a clear success/failure signal per run.

## Docker Compose + cron

```yaml
# compose.scheduled.yml
services:
  syscert:
    image: alpine:latest          # replace with your own image containing syscert
    user: "1000:1000"
    command:
      - syscert
      - --config
      - /etc/syscert/syscert.toml
      - --env-file
      - /run/secrets/syscert_secrets
    volumes:
      - ./syscert.toml:/etc/syscert/syscert.toml:ro
      - certs:/var/lib/syscert
    env_file:
      - .env
    restart: "no"                 # one-shot — the external timer re-runs it

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - certs:/etc/nginx/tls:ro
    restart: unless-stopped

volumes:
  certs:
    driver: local
```

Full annotated file:
[`examples/container/compose.scheduled.yml`](https://github.com/tfindley/syscert/blob/main/examples/container/compose.scheduled.yml)

Run it on a schedule from the host's crontab:

```sh
# crontab -e — daily at 03:17
17 3 * * * docker compose -f /opt/myapp/compose.scheduled.yml run syscert && \
           docker exec myapp-nginx-1 nginx -s reload
```

## Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: syscert
  namespace: default
spec:
  schedule: "17 3 * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            runAsGroup: 1000
            fsGroup: 1000        # ensures the PVC is writable by uid 1000
          containers:
            - name: syscert
              image: your-registry/syscert:latest
              args: [syscert, --config, /etc/syscert/syscert.toml]
              env:
                - name: CLOUDFLARE_DNS_API_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: syscert-dns-creds
                      key: CLOUDFLARE_DNS_API_TOKEN
              volumeMounts:
                - name: config
                  mountPath: /etc/syscert
                  readOnly: true
                - name: cert-store
                  mountPath: /var/lib/syscert
          volumes:
            - name: config
              configMap:
                name: syscert-config
            - name: cert-store
              persistentVolumeClaim:
                claimName: syscert-cert-store
```

Full annotated file (with PVC definition):
[`examples/container/k8s-cronjob.yaml`](https://github.com/tfindley/syscert/blob/main/examples/container/k8s-cronjob.yaml)

The app `Deployment` mounts the same PVC read-only:

```yaml
volumes:
  - name: cert-store
    persistentVolumeClaim:
      claimName: syscert-cert-store
      readOnly: true
```

## Volume ownership

In Compose, fix the volume owner before first use:

```sh
docker run --rm -v <project>_certs:/vol alpine chown 1000:1000 /vol
```

In Kubernetes, `securityContext.fsGroup` sets the group owner of a mounted PVC automatically — no
manual step needed.

## Secrets

In Compose, put credentials in a `.env` file (`0600`). In Kubernetes, store them in a `Secret` and
project them as environment variables (as shown above). Never bake secrets into the image.

## Why one-shot?

- **Rate-limit safe:** a failed run doesn't spin and hammer the CA. The next scheduled run will retry.
- **Observable:** job history gives a per-run success/failure record.
- **Restartable:** re-running the job is safe — `syscert` is idempotent.

The only advantage of `--interval` over this pattern is that the sidecar self-schedules without an
external timer. If you have cron or a Kubernetes CronJob available, this pattern is simpler.

---

Next: [Sidecar pattern](/docs/containerisation/sidecar/) ·
[Embedded pattern](/docs/containerisation/embedded/) ·
[Containerisation overview](/docs/containerisation/)
