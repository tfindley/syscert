# SysCert

[![CI](https://github.com/tfindley/syscert/actions/workflows/ci.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/ci.yml)
[![Release](https://github.com/tfindley/syscert/actions/workflows/release.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/release.yml)
[![Web](https://github.com/tfindley/syscert/actions/workflows/web.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/web.yml)

**Set-and-forget TLS for every machine.** SysCert is a small, least-privilege Linux service that
gives a host its own TLS certificate — from **Let's Encrypt** or an internal **HashiCorp Vault** /
**Smallstep `step-ca`** — keeps it renewed, and delivers it to local consumers (nginx, HAProxy,
Cockpit, databases…) with the exact ownership, mode, and SELinux context each needs. A systemd timer
keeps it fresh forever: no cron, no scripts, no cert babysitting. It's independent of any host
`certbot`.

It speaks ACME via [lego](https://go-acme.github.io/lego/) and writes certbot-compatible output
(`cert.pem` / `privkey.pem` / `chain.pem` / `fullchain.pem`) plus an all-in-one `bundle.pem`.

> **Project status: early (pre-1.0).** Working today: the full `syscert` CLI (the default *ensure*
> plus `issue` / `renew` / `distribute` / `void` / `destroy` / `dry-run` / `trust install`/`remove`),
> the systemd units, `install.sh`, and **pre-built Linux binaries** (amd64/arm64) on the
> [releases page](https://github.com/tfindley/syscert/releases). Not built yet: the Ansible role
> (see the [roadmap](docs/roadmap.md)).

## Why

You terminate public TLS at the edge (HAProxy, a load balancer). But the hop from there to your
backends — and traffic between internal services — is often plaintext, or relies on hand-made certs
that expire and page someone at 2 a.m. SysCert makes every host responsible for its own cert,
automatically:

- **Encrypt the edge→backend hop** — the backend gets its own cert, no lifecycle to manage.
- **mTLS between services** — run it on both ends against an internal CA; `syscert trust install`
  adds the CA to the system store so each side verifies the other.
- **Admin UIs & data stores** — Cockpit, Postgres, Redis, internal APIs, metrics over TLS.
- **Any host that should just have a valid, auto-renewing cert** without someone owning the renewal.

## Quick start

**Supported targets:** Debian/Ubuntu and the RHEL family (others may work but aren't tested), amd64
and arm64. One static binary and a systemd timer:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

This downloads the matching release binary, **verifies its checksum**, and runs the installer —
creating the `syscert` user, `/var/lib/syscert`, starter config + secrets, the systemd units, and
enabling (not starting) the timer. Then edit two files and you're done:

```sh
sudoedit /etc/syscert/syscert.toml      # subject, CA, challenge, distribute targets
sudoedit /etc/syscert/secrets           # e.g. CLOUDFLARE_DNS_API_TOKEN=...  (never in the .toml)

sudo -u syscert syscert dry-run --config-only                       # validate offline
sudo -u syscert syscert --staging --env-file /etc/syscert/secrets   # real run; --env-file loads creds
# happy? drop --staging, then: sudo systemctl start syscert.timer
```

The full walkthrough — complete minimal config and what each step prints — is in the
[Quick start guide](docs/quick-start.md). To inspect the script, verify checksums by hand, build from
source, or install manually, see [Advanced install](docs/advanced-install.md).

Uninstall the same way, no clone needed —
`curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall` (add `--purge` to
also remove certs/keys/config). Details in [Advanced install](docs/advanced-install.md#uninstall).

## Commands

| Command | Purpose |
|---|---|
| `syscert [--config <path>] [--staging]` | **The default.** Issue if no cert, renew if due, then distribute (what the timer runs). |
| `syscert issue [--staging]` | Obtain a fresh cert into the store. Does not distribute. |
| `syscert renew [--staging] [--force]` | Renew if due (or `--force`) into the store. Does not distribute. |
| `syscert distribute` | Copy stored artifacts to the configured targets. |
| `syscert dry-run [--config-only]` | Validate config; without `--config-only`, also run the real ACME order/challenge and discard (LE uses staging). |
| `syscert trust install` / `trust remove` | Add/remove the internal CA in the **system** trust store (root). |
| `syscert void [--force]` | Revoke the current cert, then reissue + distribute. |
| `syscert destroy [--force]` | Wipe the stored cert + ACME account (provider switch). `--keep-account` drops only the cert — reissue with no new EAB token. |
| `syscert status` | Show config + the stored cert's dates (issued/expiry/renewal), account, and distribute targets. Offline. |

`--config` defaults to `/etc/syscert/syscert.toml` (or `$SYSCERT_CONFIG`). Secrets (DNS/CA tokens)
always come from the **environment**, never the TOML, and are never logged. The systemd service
loads them from `/etc/syscert/secrets`; for a manual run, pass `--env-file /etc/syscert/secrets`
instead of exporting each one.

## Documentation

Full, canonical docs live in [`docs/`](docs/) and render on the website at
**<https://syscert.tfindley.dev/docs/>**:

- [Quick start](docs/quick-start.md) — install → edit two files → done
- [Configuration reference](docs/configuration.md) — every `syscert.toml` option
- [Sample configurations](docs/examples.md) — a starter per CA + challenge, plus annotated
  [`examples/full.toml`](examples/full.toml)
- [Advanced install](docs/advanced-install.md) — verify checksums, build from source, manual systemd
- [Distributing certs](docs/distributing.md) — artifacts, ownership/mode/SELinux, no reload hooks
- [Troubleshooting](docs/troubleshooting.md) · [FAQ](docs/faq.md) · [Roadmap](docs/roadmap.md) ·
  [Changelog](CHANGELOG.md)

## Contributing

Commits follow [Conventional Commits](https://www.conventionalcommits.org/) and releases follow
[Semantic Versioning](https://semver.org/) — see [RELEASING.md](RELEASING.md). The repo uses
[pre-commit](https://pre-commit.com/); enable the hooks once with `pre-commit install` and
`pre-commit install --hook-type commit-msg`.

## License

[AGPL-3.0](LICENSE).

## Author

[Tristan Findley](https://tfindley.co.uk). If you'd like to support the project:
[☕ Ko-fi](https://ko-fi.com/tfindley).
</content>
</invoke>
