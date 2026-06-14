---
title: Frequently asked questions
navLabel: FAQ
description: FAQ about syscert — how it differs from certbot, which CAs and challenges it supports, how it delivers certs, renewal, the trust store, security, and supported platforms.
order: 7
eyebrow: "// docs · faq"
lede: Short answers to the things people ask most. For depth, follow the links into the rest of the docs.
---

## General

### What is syscert?

A small, least-privilege Linux service that gives a host its own TLS certificate —
from Let's Encrypt or an internal HashiCorp Vault / Smallstep step-ca — keeps it
renewed via a systemd timer, and delivers it to local consumers (nginx, HAProxy,
Cockpit, databases…) with the exact ownership, mode, and SELinux context each
needs. One static binary, no daemon, no cron.

### How is it different from certbot?

syscert is independent of any host `certbot`. It speaks ACME via
[lego](https://go-acme.github.io/lego/) (a large DNS-provider set), runs as a
dedicated non-root user, and — crucially — **delivers** certs to non-root consumers
with per-target ownership/mode/SELinux instead of leaving them root-only in one
directory. It also speaks to internal CAs (Vault, step-ca), not just public ones.

## CAs & challenges

### Which CAs are supported?

Let's Encrypt (`ca = "letsencrypt"`), and any internal/other ACME CA via
`ca = "custom"` + `directory_url` — validated against HashiCorp Vault PKI and
Smallstep step-ca. See [Configuration](/docs/configuration/#which-directory_url-for-your-ca).

### Which challenge types work?

`dns-01` (default, needs no inbound ports), `http-01` and `tls-alpn-01` (CA must
reach :80/:443), and opt-in `dns-persist-01`. **Vault has no dns-01**
(http-01/tls-alpn-01 only); step-ca and Let's Encrypt support all three. Setting
`ip_sans` auto-switches to http-01/tls-alpn-01 (RFC 8738).

### What is EAB and do I need it?

External Account Binding is an out-of-band Key ID + HMAC some CAs require to
register an ACME account (Vault `eab_policy`, step-ca `requireEAB`,
ZeroSSL/Google/SSL.com). Set `[acme.eab].kid` in the config and supply the HMAC via
the `SYSCERT_EAB_HMAC` environment variable. See
[Configuration](/docs/configuration/#acmeeab--external-account-binding).

## Delivery & output

### What files does syscert produce?

Five certbot-compatible PEMs — `cert.pem`, `privkey.pem`, `chain.pem`,
`fullchain.pem` — plus a configurable all-in-one `bundle.pem`. See
[Distributing](/docs/distributing/#the-artifacts).

### Do I have to reload my services after renewal?

No — and syscert won't do it for you. It writes files and never runs commands. Have
each consumer watch its cert file and reload itself; a `systemd.path` unit is the
clean way. Example in
[Distributing → no reload hooks](/docs/distributing/#no-reload-hooks--consumers-reload-themselves).

### Can multiple services share one certificate?

Yes. Add one `[[distribute]]` block per consumer, each delivering the artifact it
needs with its own owner/group/mode.

## Renewal & lifecycle

### How often does it renew?

The timer runs daily (with jitter); bare `syscert` renews only when the cert is due.
The window is derived from the cert's lifetime by default — short-lived/IP certs
renew ~daily, long-lived use a wide window. Override with
`[renewal].renew_before = "30d"`.

### Can I renew or rotate right now?

`sudo -u syscert syscert renew --force` renews immediately. To rotate a possibly
compromised key, `syscert void` revokes then reissues. To wipe state for a provider
switch, `syscert destroy`. See [Troubleshooting → reset](/docs/troubleshooting/#reset-revoke-or-switch-providers).

## Trust store

### How do local services trust certs from an internal CA?

Run `sudo syscert trust install` once to add the CA to the system trust store
(root-only). That's separate from `acme.ca_bundle`, which trusts the CA only for the
ACME connection while bootstrapping. Public CAs are already trusted, so neither is
needed for Let's Encrypt. Details in
[Troubleshooting](/docs/troubleshooting/#x509-unknown-authority-against-an-internal-ca).

## Security & keys

### How are keys and secrets handled?

Private keys live in the store at `0600` (owned by the `syscert` user) and are
delivered with the tight mode each consumer specifies. A fresh keypair is generated
on every renewal by default (`reuse_key` opts out). Provider/CA credentials are read
from the environment — never the TOML — and are never logged.

### Does it run as root?

No. The service runs as the dedicated `syscert` user with only `CAP_CHOWN` (plus
`CAP_NET_BIND_SERVICE` if you serve http-01/tls-alpn-01), under a hardened systemd
unit. Only `syscert trust install/remove` needs root.

## Platforms

### What's supported?

Debian/Ubuntu and the RHEL family (others may work but aren't tested), on Linux
amd64 and arm64. The host needs systemd.

### Is there an Ansible role?

Not yet — it's on the [roadmap](/docs/roadmap/) for fleet installs and will perform
the same steps as `install.sh`. syscert is pre-1.0; expect rough edges and please
[report issues](https://github.com/tfindley/syscert/issues).

---

Next: [Quick start](/docs/quick-start/) · [Troubleshooting](/docs/troubleshooting/)
