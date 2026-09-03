---
title: Troubleshooting
navLabel: Troubleshooting
description: Fixes for the common syscert problems — no FQDN, untrusted internal CAs, distribution permissions, stale certs after renewal, timer scheduling, and how to test safely.
order: 8
eyebrow: "// docs · troubleshooting"
lede: The failure modes you're most likely to hit, what causes them, and the fix.
---

## Where to look first

Operational logs go to stderr (the journal under systemd); command results go to stdout. Start here:

```sh
journalctl -u syscert.service -f                 # follow the service logs
sudo -u syscert syscert dry-run --config-only    # what config is actually resolved
```

To turn up verbosity, including lego's ACME detail, raise the log level:

```toml
[logging]
level = "debug"   # includes lego's ACME detail
```

## "no FQDN" — refuses to run

syscert builds the certificate around the host's FQDN and **never guesses**. If the host has no fully-qualified domain name it errors out. Fix it by setting `cert.hostname = "host.example.com"` in the config, or by giving the host a proper FQDN.

## "x509: unknown authority" against an internal CA

Requesting from Vault or step-ca makes an HTTPS call to the CA's ACME endpoint, and Go verifies that against the **system** trust store. If the host doesn't trust the internal CA yet, the call fails before anything else happens. Classic chicken-and-egg. Two trust settings solve two separate problems:

- **For the ACME connection only:** set `acme.ca_bundle = "/etc/syscert/internal-ca.pem"` to trust the CA just for that request (no host changes). syscert warns when it's set.
- **For the whole system:** once the cert is installed, run `sudo syscert trust install` to add the CA root/intermediates to the system trust store, so other local consumers and clients trust the issued certs. Undo with `sudo syscert trust remove`.

Let's Encrypt needs neither; the system already trusts it.

## Distribution can't write to a target

Confirm the target directory exists and the `syscert` user can write to it. When the target is owned by a different user, the copy needs `CAP_CHOWN`, which the shipped systemd unit grants (`AmbientCapabilities=CAP_CHOWN`). A **rejected world-readable mode** on `privkey`/`bundle` is intentional: key-bearing artifacts have to use a tight mode such as `0600`.

If the certificate is issued and only the *copy* fails — typically with a **read-only file system** error — the cause is the unit's own sandbox rather than permissions. `ProtectSystem=strict` makes the whole filesystem read-only, and the shipped unit can only name the store in `ReadWritePaths`, because a static unit can't know your targets. Grant each target's directory with a drop-in:

```sh
sudo systemctl edit syscert.service
```

```ini
[Service]
ReadWritePaths=/etc/nginx/tls
```

It works by hand and fails under the timer precisely because a manual run isn't sandboxed. The [Ansible role](/docs/advanced-install/ansible/) derives this list from your `syscert_distribute` targets, so it doesn't arise there.

## A service still serves the old certificate after renewal

syscert delivers files but **never reloads consumers**. The service has to watch its cert file and reload itself, and the clean way is a `systemd.path` unit. See [Reloading services](/docs/reloading/) for the pattern and per-service reload commands.

## The timer's "next run" is in the future / nothing happened

That's normal. The timer fires shortly after boot and daily with jitter, but bare `syscert` only renews when the certificate is actually due. A run where nothing's due does nothing. Check scheduling with `systemctl list-timers syscert.timer`, and the last run with `journalctl -u syscert.service`. Remember that `install.sh` **enables** the timer but doesn't **start** it, so run `sudo systemctl start syscert.timer` once after you've configured things.

## Testing safely before production

Validate offline with `syscert dry-run --config-only` (no network). The full `syscert dry-run` runs a real ACME order and challenge, then discards the cert; against Let's Encrypt it uses staging automatically. Add `--staging` to `issue`/`renew`/`void`/bare `syscert` to route Let's Encrypt to staging during a real run.

Running by hand (not via the systemd unit)? The unit loads DNS/CA credentials from `/etc/syscert/secrets`; a manual run won't, so either export them or pass `--env-file /etc/syscert/secrets` (repeatable; the existing environment wins).

## IP-SAN and Vault gotchas

- Private IP plus public CA gets rejected. A private (RFC 1918) IP SAN requires an internal CA; public-CA IP certs need a public IP and `acme.profile = "shortlived"`.
- IP SANs force http-01/tls-alpn-01 (RFC 8738 forbids dns-01 for IPs), so the CA has to reach the host on :80/:443. Open the firewall.
- Vault's ACME has a known IPv6 challenge issue, so prefer IPv4 in the directory URL and IP SANs for now.

## "store is owned by …" — wrong user running syscert

syscert refuses early if the running user doesn't match the store's owner, rather than creating files the scheduled timer can't later renew or overwrite. Two cases:

- **Running as root over a `syscert`-owned store.** If the store at `/var/lib/syscert` is owned by the `syscert` user and you invoke `sudo syscert …`, the command fails:

  > store /var/lib/syscert is owned by syscert; running as root would create files
  > syscert can't renew — run as that user: `sudo -u syscert syscert …`

Use `sudo -u syscert syscert …` (not bare `sudo`) for all write operations. The systemd timer already runs as `User=syscert`, so the service is unaffected.

- **Running as an unprivileged user who doesn't own the store.** If you try to run syscert as a user other than the store's owner, the command fails:

  > store /var/lib/syscert is owned by syscert; run syscert as that user or root
  > (the systemd timer does this for you)

Switch to the correct user (`sudo -u syscert syscert …`) or let the timer handle it.

Read-only commands (`status`, `dry-run`, `version`, `trust`) are unaffected, and a store that does not yet exist is not checked.

## SELinux: binary not executable under systemd (RHEL enforcing)

On an **enforcing RHEL / CentOS / Rocky / AlmaLinux** host, a binary installed from a non-standard location like `/root` or a home directory keeps the source label (often `admin_home_t`), and `systemd` (`init_t`) can't execute it. You'll see it as the timer failing with a permission denial in the audit log.

`install.sh` handles this automatically: after placing the binary at `/usr/local/bin/syscert` it runs `restorecon -R /usr/local/bin/syscert` (alongside the store and config dirs), which gives the binary `bin_t` so the timer can execute it. No custom policy module is needed.

If you placed the binary yourself without the installer, one command fixes it:

```sh
sudo restorecon /usr/local/bin/syscert
```

Verify the label afterwards with `ls -Z /usr/local/bin/syscert`; it should show `system_u:object_r:bin_t:s0`.

## Reset, revoke, or switch providers

To rotate a possibly-compromised key, or to tear down and switch CAs:

```sh
sudo -u syscert syscert status                  # inspect the current cert + account state first
sudo -u syscert syscert void --force            # revoke, then reissue + distribute (key compromised)
sudo -u syscert syscert destroy --keep-account  # drop the cert, KEEP the account (reissue, no new EAB token)
sudo -u syscert syscert destroy --force         # wipe stored cert + ACME account (switching CA)
```

`status` is read-only: it shows the config, the stored cert's issue/expiry/renewal dates, the account, and the distribute targets. `void` revokes (if the CA supports it) then reissues. `destroy --keep-account` removes only the certificate and any archived snapshots, so the next run reissues **using the existing account, with no new EAB token needed**. Plain `destroy` also wipes the ACME account (and can un-trust an internal CA), but it does **not** revoke or reissue; after it, update the config and run `syscert issue`.

---

Next: [Configuration](/docs/configuration/) · [FAQ](/docs/faq/)
