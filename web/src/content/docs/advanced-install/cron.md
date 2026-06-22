---
title: Install as a cron job
navLabel: As a cron job
description: Run syscert on appliances and NAS boxes without systemd (e.g. Asustor ADM) by scheduling `syscert` from cron — with --env-file for secrets and a low-privilege user where possible.
order: 3
eyebrow: "// docs · advanced install · cron"
lede: No systemd? On an appliance or NAS, schedule syscert from cron. The binary is the same — only the scheduler changes.
---

SysCert's normal scheduler is a [systemd timer](/docs/advanced-install/manually/#the-user-service-and-timer),
but the binary doesn't need systemd: it's a single static binary that issues, renews,
and distributes on each run, then exits. On appliances and NAS devices that lack
systemd — Asustor ADM and similar BusyBox/cron-based systems — run it from **cron**
instead. Everything else (config, store, distribution) is unchanged.

## 1. Place the binary

Download and verify a release binary as in [Manually → Download a release binary &
verify](/docs/advanced-install/manually/#download-a-release-binary--verify), then put
it somewhere **persistent**. NAS firmware updates often wipe `/usr/local`, so prefer a
path on a data volume:

```sh
install -m 0755 ./syscert /volume1/syscert/syscert
```

## 2. Config and secrets

Keep the config and secrets on the same persistent volume:

- `/volume1/syscert/syscert.toml` — your configuration. Point SysCert at it with
  `--config` (below) or `SYSCERT_CONFIG`.
- `/volume1/syscert/secrets` — DNS/CA credentials, **`chmod 0600`** so only the user
  cron runs as can read it. Secrets never go in the TOML.

## 3. Schedule it

Cron runs with a **minimal environment**, so load the secrets explicitly with
`--env-file` and use absolute paths everywhere:

```sh
# crontab -e — run daily at 03:17 (pick an odd minute so you're not on the hour)
17 3 * * * /volume1/syscert/syscert --config /volume1/syscert/syscert.toml --env-file /volume1/syscert/secrets >> /volume1/syscert/syscert.log 2>&1
```

Bare `syscert` runs the default **ensure** action — issue if missing, renew if due, then distribute — and
a **no-op when nothing is due**, so a daily run is cheap and safe. `--env-file` loads
the credentials the systemd unit would otherwise get from `/etc/syscert/secrets`; an
existing environment variable always wins, and values are never logged.

On Asustor ADM (and many NAS web UIs) you don't have to touch `crontab` directly — add
the same command as a daily **user-defined script** in the built-in task scheduler.

## Run as a low-privilege user

Where the appliance allows it, run the job as a dedicated, non-root user that owns the
config, secrets, and store — not root. If you must run as root, keep the secrets file
`0600` and the store directory `0700`. SysCert only needs `CAP_CHOWN` when it must set
a *different* owner on a distributed copy; if it runs as the same user that consumes
the certificate, no elevated capabilities are required.

**The user running syscert must own the store.** If the store was created by one user
and you later invoke syscert as a different user — including root over a
non-root-owned store — syscert refuses early rather than creating files the original
owner can't renew. Always run syscert as the same user that owns the store directory.

## No reload hooks, no journal

SysCert never restarts your services — each consumer watches its own certificate file
and reloads itself (see [Reloading services](/docs/reloading/)). And without journald,
send output to a log file (as above) and rotate it with the appliance's own tools.

## Uninstall

There's nothing system-level to undo: remove the crontab line (`crontab -e`) or the
scheduled task, then delete the binary, config, secrets, and the store directory you
created.

---

Next: [Configuration](/docs/configuration/) · [Reloading services](/docs/reloading/) ·
[Manually](/docs/advanced-install/manually/)
