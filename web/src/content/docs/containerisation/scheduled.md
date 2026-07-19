---
title: Scheduled pattern
navLabel: Scheduled
description: Run syscert as a one-shot job on an external timer — the simplest container pattern, and the Kubernetes-native path via CronJob.
order: 2
eyebrow: "// docs · containerisation · scheduled"
lede: The simplest pattern. syscert runs once, exits, and an external timer brings it back. Mirrors the systemd-timer model and maps directly to a Kubernetes CronJob.
---

For most setups, the scheduled pattern is **recommended**. `syscert` runs without `--interval`, does one ensure cycle (issue if missing, renew if due, distribute), and exits. Something external re-runs it on a schedule: cron, a systemd timer on the host, or a Kubernetes CronJob.

It mirrors the host systemd-timer model: a one-shot binary on a schedule. Easy to inspect, safe to restart, and every run gives you a clear success or failure signal.

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

Full annotated file: [`examples/container/compose.scheduled.yml`](https://github.com/tfindley/syscert/blob/main/examples/container/compose.scheduled.yml)

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

Full annotated file (with PVC definition): [`examples/container/k8s-cronjob.yaml`](https://github.com/tfindley/syscert/blob/main/examples/container/k8s-cronjob.yaml)

The app `Deployment` mounts the same PVC read-only:

```yaml
volumes:
  - name: cert-store
    persistentVolumeClaim:
      claimName: syscert-cert-store
      readOnly: true
```

## Volume ownership

In Compose, fix the volume owner before you use it:

```sh
docker run --rm -v <project>_certs:/vol alpine chown 1000:1000 /vol
```

In Kubernetes, `securityContext.fsGroup` sets the group owner of a mounted PVC for you, so there's no manual step.

## Secrets

In Compose, put credentials in a `.env` file (`0600`). In Kubernetes, keep them in a `Secret` and project them as environment variables, like the example above. Never bake secrets into the image.

## Why one-shot?

A few things make one-shot nice. A failed run doesn't spin and hammer the CA; the next scheduled run just retries, so you stay rate-limit safe. Job history hands you a success or failure record per run. And re-running the job is safe, because `syscert` is idempotent.

The one thing `--interval` buys you over this pattern is that the sidecar schedules itself without an external timer. If you've already got cron or a Kubernetes CronJob, this pattern is simpler.

---

Next: [Sidecar pattern](/docs/containerisation/sidecar/) · [Embedded pattern](/docs/containerisation/embedded/) · [Containerisation overview](/docs/containerisation/)
