---
title: Reloading services on renewal
navLabel: Reloading services
description: SysCert never runs reload hooks; each consumer watches its cert and reloads itself. The systemd.path pattern, plus the reload command for common services.
order: 7
eyebrow: "// docs · reloading"
lede: SysCert delivers files and gets out of the way; it never restarts your services. This is how to have each consumer pick up a renewed certificate itself, the clean systemd way.
---

SysCert writes files and **never runs commands**: no reloads, no restarts, no post-hooks.
That keeps this least-privilege service from having to poke at arbitrary daemons (see
[Distributing](/docs/distributing/) for the delivery model). The flip side is that **you** wire each
consumer to pick up its new certificate.

## Most services do *not* auto-reload

People often assume nginx or Apache will notice a changed cert file on their own. They
don't. Both read the certificate and key **once**, at start or reload, and hold them in
memory. After a renewal overwrites the files, the old cert keeps being served until the
service is told to reload. Same story for HAProxy, Postfix, Dovecot, and most others. A few
servers *do* watch their cert files (Caddy and Traefik, which run their own ACME), but
they're the exception.

So for almost everything, you need a small nudge after each renewal.

## The pattern: a `systemd.path` watcher

Have systemd watch the cert file SysCert delivers, and run a reload when it changes. That's
two units per service: a `.path` that watches, and a oneshot `.service` that reloads:

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

Enable the **`.path`**, not the service; it starts the service on each change:

```sh
sudo systemctl enable --now nginx-reload.path
```

Now every time SysCert re-delivers `fullchain.pem`, systemd reloads nginx, with no
privileged hook inside SysCert.

- **Watch the path the consumer actually reads:** your `[[distribute]]` target, not the
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
`restart` instead. The `.path` mechanism is identical either way.

---

Next: [Distributing certs](/docs/distributing/) · [Configuration](/docs/configuration/)
