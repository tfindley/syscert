---
title: Distributing certificates
navLabel: Distributing certs
description: How syscert delivers certificates — the canonical store, the five certbot-compatible artifacts plus bundle.pem, per-target ownership/mode/SELinux, and why there are no reload hooks.
order: 6
eyebrow: "// docs · distributing"
lede: syscert keeps one source of truth and copies the pieces each consumer needs — with the exact ownership, mode, and SELinux context — then gets out of the way.
---

## The canonical store

Every issuance and renewal writes to one place: the canonical store at `/var/lib/syscert` (owned by the `syscert` user, `0700`; key-bearing files `0600`). That store is the source of truth. Distribution is a separate step that **copies** artifacts out to consumers instead of pointing every service at one shared directory. Each renewal re-copies the files and re-applies ownership, mode, and SELinux context.

## The artifacts

Per certificate, syscert writes five PEM files with certbot-compatible names:

| Artifact | Contents | Holds key? |
|---|---|---|
| `cert` (cert.pem) | leaf certificate only | no |
| `privkey` (privkey.pem) | private key | **yes** |
| `chain` (chain.pem) | intermediate chain (no leaf, no root) | no |
| `fullchain` (fullchain.pem) | leaf + intermediates (what most servers want) | no |
| `bundle` (bundle.pem) | configurable all-in-one (default leaf + chain + root + key) | **yes** |

The first four come straight from the ACME response. The **root** in `bundle.pem` only comes from internal CAs like Vault or step-ca; public CAs don't provide one, so it's left out. Compose the bundle with `[bundle].order`, described in [Configuration](/docs/configuration/#bundle--all-in-one-file).

## Delivery targets

Each `[[distribute]]` block copies **one artifact** to a path with the ownership, mode, and (optionally) SELinux context that consumer needs. Writes are atomic. Key-bearing artifacts (`privkey`, `bundle`) can't be world-readable; a permissive mode gets rejected up front. Add as many blocks as you have consumers:

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

syscert writes files, and it **never runs commands**: no reloads, no restarts, no hooks. That keeps this least-privilege service from having to poke at arbitrary daemons. Instead, each consumer watches its cert file and reloads itself, and a small `systemd.path` unit is the clean way to do it. See **[Reloading services](/docs/reloading/)** for the pattern and the reload command per service.

---

Next: [Configuration → distribute](/docs/configuration/#distribute--delivering-to-consumers) · [Troubleshooting](/docs/troubleshooting/)
