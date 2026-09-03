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

## Privileged target directories

Most interesting targets are directories another package owns — `/etc/cockpit/ws-certs.d`, `/etc/nginx/tls`, `/etc/pki/…`. Two things stand between the service and a file there, and both bite *after* a successful issuance: the certificate is obtained, then can't be delivered. The failure looks like a bare `read-only file system` or `permission denied` in the journal, which is why syscert now names the remedy in the error itself.

**First, the sandbox.** The unit runs under `ProtectSystem=strict`, so the whole filesystem is read-only except what `ReadWritePaths=` grants. A static unit can't know your targets, so syscert derives the list from your config:

```sh
sudo syscert systemd-paths --write     # writes /etc/systemd/system/syscert.service.d/10-distribute-paths.conf
sudo systemctl daemon-reload
```

Run `syscert systemd-paths` without `--write` to see the file first. Re-run it after adding or moving a target — [`install.sh`](/docs/advanced-install/manually/) does this for you on every run, and the [Ansible role](/docs/advanced-install/ansible/) derives it from `syscert_distribute`.

**Second, ordinary permissions.** syscert runs as a non-root user with `CAP_CHOWN` and nothing else. Creating a file needs write and execute on the *directory*, and `CAP_CHOWN` does not grant that — it only lets syscert set the owner of a file it has already created, which is how a delivered file still ends up `root:root 0644`. A directory like `/etc/cockpit/ws-certs.d`, typically `root:root 0755`, therefore refuses it. Grant that one user on that one directory:

```sh
sudo setfacl -m u:syscert:rwx /etc/cockpit/ws-certs.d
```

An ACL is preferable to `chgrp syscert … && chmod g+w …` because it leaves the owner and group exactly as the owning package set them, so nothing else notices. Both are fine; pick the ACL unless the filesystem doesn't support one. `install.sh` and the Ansible role apply this for you; the manual route is the command above.

Test through the timer, not by hand — `sudo systemctl start syscert.service` then `journalctl -u syscert -n 30`. An interactive `sudo -u syscert syscert distribute` isn't sandboxed, so it exercises the permissions but not the first barrier, and can pass while the timer still fails.

If a target is still unwritable, syscert delivers to every *other* target anyway and then exits non-zero: one misconfigured consumer doesn't deny the rest their renewed certificate.

## No reload hooks — consumers reload themselves

syscert writes files, and it **never runs commands**: no reloads, no restarts, no hooks. That keeps this least-privilege service from having to poke at arbitrary daemons. Instead, each consumer watches its cert file and reloads itself, and a small `systemd.path` unit is the clean way to do it. See **[Reloading services](/docs/reloading/)** for the pattern and the reload command per service.

---

Next: [Configuration → distribute](/docs/configuration/#distribute--delivering-to-consumers) · [Troubleshooting](/docs/troubleshooting/)
