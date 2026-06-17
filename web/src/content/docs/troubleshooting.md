---
title: Troubleshooting
navLabel: Troubleshooting
description: Fixes for the common syscert problems — no FQDN, untrusted internal CAs, distribution permissions, stale certs after renewal, timer scheduling, and how to test safely.
order: 7
eyebrow: "// docs · troubleshooting"
lede: The failure modes you're most likely to hit, what causes them, and the fix.
---

## Where to look first

Operational logs go to stderr (the journal under systemd); command results go to
stdout. Start here:

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

syscert builds the certificate around the host's FQDN and **never guesses**. If the
host has no fully-qualified domain name it errors out. Fix it by setting
`cert.hostname = "host.example.com"` in the config, or by giving the host a proper
FQDN.

## "x509: unknown authority" against an internal CA

Requesting from Vault/step-ca makes an HTTPS call to the CA's ACME endpoint, which
Go verifies against the **system** trust store. If the host doesn't trust the
internal CA yet, that call fails before anything happens — a chicken-and-egg. Two
distinct trust settings solve two distinct problems:

- **For the ACME connection only:** set
  `acme.ca_bundle = "/etc/syscert/internal-ca.pem"` to trust the CA just for that
  request (no host changes). syscert warns when it's set.
- **For the whole system:** once the cert is installed, run
  `sudo syscert trust install` to add the CA root/intermediates to the system trust
  store, so other local consumers and clients trust the issued certs. Undo with
  `sudo syscert trust remove`.

Let's Encrypt needs neither — it's already trusted by the system.

## Distribution can't write to a target

Confirm the target directory exists and is writable by the `syscert` user. When the
target is owned by a different user, the copy needs `CAP_CHOWN` — the shipped
systemd unit grants it (`AmbientCapabilities=CAP_CHOWN`). A **rejected
world-readable mode** on `privkey`/`bundle` is intentional: key-bearing artifacts
must use a tight mode such as `0600`.

## A service still serves the old certificate after renewal

syscert delivers files but **never reloads consumers**. The service must watch its
cert file and reload itself — the clean way is a `systemd.path` unit. See
[Reloading services](/docs/reloading/) for the pattern and per-service reload commands.

## The timer's "next run" is in the future / nothing happened

That's normal: the timer fires shortly after boot and daily with jitter, but bare
`syscert` only renews when the certificate is actually due. A run where nothing was
due is a no-op. Check scheduling with `systemctl list-timers syscert.timer` and the
last run with `journalctl -u syscert.service`. Remember `install.sh` **enables** but
doesn't **start** the timer — run `sudo systemctl start syscert.timer` once after
configuring.

## Testing safely before production

Validate offline with `syscert dry-run --config-only` (no network). The full
`syscert dry-run` performs a real ACME order + challenge and then discards the cert
— Let's Encrypt automatically uses staging. Add `--staging` to
`issue`/`renew`/`void`/bare `syscert` to route Let's Encrypt to staging during a
real run.

Running by hand (not via the systemd unit)? The unit loads DNS/CA credentials from
`/etc/syscert/secrets`; a manual run won't, so either export them or pass
`--env-file /etc/syscert/secrets` (repeatable; the existing environment wins).

## IP-SAN and Vault gotchas

- **Private IP + public CA is rejected.** A private (RFC 1918) IP SAN requires an
  internal CA; public-CA IP certs need a public IP and `acme.profile = "shortlived"`.
- **IP SANs force http-01/tls-alpn-01** (RFC 8738 forbids dns-01 for IPs), so the CA
  must reach the host on :80/:443 — open the firewall.
- **Vault's ACME has a known IPv6 challenge issue** — prefer IPv4 in the directory
  URL and IP SANs for now.

## Reset, revoke, or switch providers

To rotate a possibly-compromised key, or to tear down and switch CAs:

```sh
sudo -u syscert syscert status                  # inspect the current cert + account state first
sudo -u syscert syscert void --force            # revoke, then reissue + distribute (key compromised)
sudo -u syscert syscert destroy --keep-account  # drop the cert, KEEP the account (reissue, no new EAB token)
sudo -u syscert syscert destroy --force         # wipe stored cert + ACME account (switching CA)
```

`status` is read-only — config, the stored cert's issue/expiry/renewal dates, the
account, and distribute targets. `void` revokes (if the CA supports it) then reissues.
`destroy --keep-account` removes only the certificate (and any archived snapshots),
so the next run reissues **reusing the existing account — no new EAB token needed**.
Plain `destroy` also wipes the ACME account (and can un-trust an internal CA) but does
**not** revoke or reissue; after it, update the config and run `syscert issue`.

---

Next: [Configuration](/docs/configuration/) · [FAQ](/docs/faq/)
