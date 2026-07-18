---
title: Install as a cron job
navLabel: As a cron job
description: Run syscert on appliances and NAS boxes without systemd (e.g. Asustor ADM) by scheduling `syscert` from cron — with --env-file for secrets and a low-privilege user where possible.
order: 3
eyebrow: "// docs · advanced install · cron"
lede: No systemd? On an appliance or NAS, schedule syscert from cron. The binary is the same — only the scheduler changes.
---

SysCert usually schedules itself with a [systemd timer](/docs/advanced-install/manually/#the-user-service-and-timer),
but the binary doesn't need systemd at all. It's one static binary that issues and renews
certs, distributes them, then exits on each run. On appliances and NAS boxes with no
systemd (Asustor ADM and other BusyBox/cron systems), run it from **cron** instead.
Config, store, distribution: none of that changes.

## 1. Place the binary

Download and verify a release binary the same way as in [Manually → Download a release binary &
verify](/docs/advanced-install/manually/#download-a-release-binary--verify), then put
it somewhere **persistent**. NAS firmware updates like to wipe `/usr/local`, so pick a
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

Bare `syscert` runs the default **ensure** action: issue if the cert's missing, renew if it's
due, then distribute. When nothing's due it's a **no-op**, so a daily run costs almost nothing.
`--env-file` loads the credentials the systemd unit would otherwise pull from `/etc/syscert/secrets`.
An existing environment variable always wins, and values never reach the log.

On Asustor ADM (and plenty of NAS web UIs) you don't have to touch `crontab` at all. Add
the same command as a daily **user-defined script** in the built-in task scheduler.

## Run as a low-privilege user

Where the appliance lets you, run the job as a dedicated non-root user that owns the
config, secrets, and store. Not root. If you're stuck running as root, keep the secrets
file `0600` and the store directory `0700`. SysCert only wants `CAP_CHOWN` when it has to
set a *different* owner on a distributed copy. Run it as the same user that consumes the
certificate and it needs no elevated capabilities at all.

**The user running syscert must own the store.** Say one user created the store and you
later run syscert as another (root over a non-root store counts), syscert bails out early
rather than write files the original owner could never renew. Always run syscert as the
user that owns the store directory.

## Alternative: `--interval` instead of crond

If the appliance runs a container, or its cron daemon isn't dependable, the `--interval` flag
replaces the external scheduler. `syscert --interval 12h` runs the ensure loop in-process:

```sh
/volume1/syscert/syscert --interval 12h --config /volume1/syscert/syscert.toml \
  --env-file /volume1/syscert/secrets >> /volume1/syscert/syscert.log 2>&1 &
```

Start it from an init script or a startup hook and the process schedules itself. `SIGTERM` shuts
it down cleanly: it finishes the current cycle, then exits. The minimum interval is `1m`, and the
`SYSCERT_INTERVAL` environment variable does the same job as the flag.

It's the same model [container setups](/docs/containerisation/) use when there's no cron.
Stick with the cron approach above when the appliance already has a task scheduler you trust.

## No reload hooks, no journal

SysCert never restarts your services. Each consumer watches its own certificate file and
reloads itself (see [Reloading services](/docs/reloading/)). With no journald around,
send output to a log file like the example above and rotate it with the appliance's own tools.

## Uninstall

Nothing system-level to undo here. Remove the crontab line (`crontab -e`) or the
scheduled task, then delete the binary, config, secrets, and the store directory you
made.

---

Next: [Configuration](/docs/configuration/) · [Reloading services](/docs/reloading/) ·
[Manually](/docs/advanced-install/manually/)
