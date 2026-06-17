---
title: Reloading services on renewal
navLabel: Reloading services
description: SysCert never runs reload hooks — each consumer watches its cert and reloads itself. The systemd.path pattern, plus the reload command for common services.
order: 6
eyebrow: "// docs · reloading"
lede: SysCert delivers files and gets out of the way — it never restarts your services. Here's how to have each consumer pick up a renewed certificate itself, the clean systemd way.
---

SysCert writes files and **never runs commands** — no reloads, restarts, or post-hooks.
That keeps the least-privilege service from needing to poke at arbitrary daemons (see
[Distributing](/docs/distributing/) for the delivery model). The flip side: **you** wire each
consumer to pick up its new certificate.

## Most services do *not* auto-reload

A common assumption is that nginx or Apache notice a changed cert file on their own. They
don't — both read the certificate and key **once**, at start or reload, and keep them in
memory. After a renewal overwrites the files, the old cert is served until the service is
told to reload. The same is true of HAProxy, Postfix, Dovecot, and most others. A few
servers *do* watch their cert files (e.g. **Caddy** and **Traefik**, which manage their own
ACME), but they're the exception.

So for almost everything, you need a small nudge after each renewal.

## The pattern: a `systemd.path` watcher

Have systemd watch the cert file SysCert delivers, and run a reload when it changes. Two
units per service — a `.path` that watches, and a oneshot `.service` that reloads:

```ini
# /etc/systemd/system/nginx-reload.path
[Path]
PathChanged=/etc/nginx/tls/fullchain.pem   # the path SysCert distributes to

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/nginx-reload.service
[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl reload nginx
```

Enable the **`.path`** (not the service) — it starts the service on each change:

```sh
sudo systemctl enable --now nginx-reload.path
```

Now every time SysCert re-delivers `fullchain.pem`, systemd reloads nginx — with no
privileged hook inside SysCert.

- **Watch the path the consumer actually reads** — your `[[distribute]]` target, not the
  central store (which the service can't read anyway).
- `PathChanged=` fires when the file is closed after writing (good for SysCert's atomic
  replace); `PathModified=` is noisier. To watch a whole directory, point `PathModified=` at
  it or use `DirectoryNotEmpty=`.

## Reload command per service

Swap the `ExecStart=` (or the watched path) to match the consumer:

| Service | Reload command | Notes |
|---|---|---|
| nginx | `systemctl reload nginx` | or `nginx -s reload` (SIGHUP) |
| Apache (httpd) | `systemctl reload httpd` | `apachectl graceful` |
| HAProxy | `systemctl reload haproxy` | graceful; needs the key+cert in one PEM — use a `bundle` artifact |
| Postfix | `systemctl reload postfix` | re-reads `smtpd_tls_*` certs |
| Dovecot | `systemctl reload dovecot` | |
| Cockpit | `systemctl restart cockpit` | reload not supported; restart is quick |
| PostgreSQL | `systemctl reload postgresql` | or `SELECT pg_reload_conf();` — re-reads `ssl_cert_file`/`ssl_key_file` |

If a daemon can't reload TLS material without a full restart, point the `.service` at
`restart` instead — the `.path` mechanism is identical.

---

Next: [Distributing certs](/docs/distributing/) · [Configuration](/docs/configuration/)
