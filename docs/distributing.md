---
title: Distributing certificates
navLabel: Distributing certs
description: How syscert delivers certificates — the canonical store, the five certbot-compatible artifacts plus bundle.pem, per-target ownership/mode/SELinux, and why there are no reload hooks.
order: 5
eyebrow: "// docs · distributing"
lede: syscert keeps one source of truth and copies the pieces each consumer needs — with the exact ownership, mode, and SELinux context — then gets out of the way.
---

## The canonical store

Every issuance and renewal writes to one place: the canonical store at
`/var/lib/syscert` (owned by the `syscert` user, `0700`; key-bearing files
`0600`). That store is the source of truth — distribution is a separate step that
**copies** artifacts out to consumers, rather than pointing every service at one
shared directory. Each renewal re-copies and re-applies ownership, mode, and
SELinux context.

## The artifacts

Per certificate, syscert writes five PEM files with certbot-compatible names:

| Artifact | Contents | Holds key? |
|---|---|---|
| `cert` (cert.pem) | leaf certificate only | no |
| `privkey` (privkey.pem) | private key | **yes** |
| `chain` (chain.pem) | intermediate chain (no leaf, no root) | no |
| `fullchain` (fullchain.pem) | leaf + intermediates (what most servers want) | no |
| `bundle` (bundle.pem) | configurable all-in-one (default leaf + chain + root + key) | **yes** |

The first four come straight from the ACME response. The **root** in `bundle.pem`
is only available from internal CAs (Vault/step-ca); for public CAs it's omitted.
Compose the bundle with `[bundle].order` — see
[Configuration](/docs/configuration/#bundle--all-in-one-file).

## Delivery targets

Each `[[distribute]]` block copies **one artifact** to a path with the ownership,
mode, and (optionally) SELinux context that consumer needs. Writes are atomic.
Key-bearing artifacts (`privkey`, `bundle`) must not be world-readable — a
permissive mode is rejected up front. Add as many blocks as you have consumers:

```toml
# nginx wants the fullchain + key
[[distribute]]
artifact = "fullchain"
path     = "/etc/nginx/tls/fullchain.pem"
owner    = "root"
group    = "root"
mode     = "0644"

[[distribute]]
artifact = "privkey"
path     = "/etc/nginx/tls/privkey.pem"
owner    = "root"
group    = "root"
mode     = "0600"

# an app that wants one all-in-one file, owned by its own user
[[distribute]]
artifact        = "bundle"
path            = "/etc/someapp/tls/combined.pem"
owner           = "someapp"
group           = "someapp"
mode            = "0600"
selinux_context = "cert_t"
```

> Delivering to a path owned by another user needs `CAP_CHOWN`, which the shipped
> unit grants. On the RHEL family, set `selinux_context` (e.g. `cert_t`) so the
> consumer's domain can read the file; syscert relabels after writing.

## No reload hooks — consumers reload themselves

syscert writes files and **never runs commands** — it issues no reloads, restarts,
or hooks. This keeps the least-privilege service from needing to poke at arbitrary
daemons. Instead, have each consumer watch its cert file and reload itself. A small
`systemd.path` unit is the clean way to do it:

```ini
# /etc/systemd/system/nginx-reload.path
[Path]
PathChanged=/etc/nginx/tls/fullchain.pem

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/nginx-reload.service
[Service]
Type=oneshot
ExecStart=/bin/systemctl reload nginx
```

Enable with `sudo systemctl enable --now nginx-reload.path`. Now whenever syscert
re-delivers `fullchain.pem`, systemd reloads nginx — with no privileged hook inside
syscert. Many servers (e.g. HAProxy with certificate watching, or anything behind a
`systemd.path`) can do the same.

---

Next: [Configuration → distribute](/docs/configuration/#distribute--delivering-to-consumers) ·
[Troubleshooting](/docs/troubleshooting/)
