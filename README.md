# SysCert

[![CI](https://github.com/tfindley/syscert/actions/workflows/ci.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/ci.yml) [![Release](https://github.com/tfindley/syscert/actions/workflows/release.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/release.yml) [![Web](https://github.com/tfindley/syscert/actions/workflows/web.yml/badge.svg)](https://github.com/tfindley/syscert/actions/workflows/web.yml)

**Every machine gets its own TLS certificate, then forgets about it.** SysCert is a small, least-privilege Linux service. It gives a host a certificate (from **Let's Encrypt**, or an internal CA like **HashiCorp Vault** or **Smallstep `step-ca`**), renews it before it lapses, and drops it where local consumers actually read it — nginx, HAProxy, Cockpit, a database — with the exact owner, mode, and SELinux context each one needs. A systemd timer keeps it current: no cron, no renewal scripts to babysit. And it doesn't touch, or need, a host `certbot`.

Under the hood it speaks ACME through [lego](https://go-acme.github.io/lego/) and writes the certbot-compatible files you already expect (`cert.pem`, `privkey.pem`, `chain.pem`, `fullchain.pem`), plus one `bundle.pem` with the lot in it.

> **Project status: early (pre-1.0).** Working today: the full `syscert` CLI (the default *ensure*
> plus `issue` / `renew` / `distribute` / `void` / `destroy` / `dry-run` / `status` /
> `trust install`/`remove` / `systemd-paths`),
> the systemd units, `install.sh`, **pre-built Linux binaries** (amd64/arm64) on the
> [releases page](https://github.com/tfindley/syscert/releases), and an
> [Ansible role](docs/advanced-install/ansible.md) for fleet installs.

## Why

You terminate public TLS at the edge — HAProxy, a load balancer, whatever sits out front. The hop from there to your backends, though, is usually plaintext, and so is the chatter between internal services. When it isn't, it's a hand-rolled cert that expires on a Sunday and pages someone at 2 a.m. SysCert hands that job to each host. The box owns its cert, and that's the end of it.

Where it earns its keep:

- Encrypting the edge-to-backend hop. The backend carries its own cert, with no separate lifecycle to track.
- mTLS between services. Run SysCert on both ends against an internal CA, and `syscert trust install` puts that CA in the system store so each side can verify the other.
- TLS on the admin and data plane: Cockpit, Postgres, Redis, internal APIs, metrics.
- Or just any host that ought to have a valid, self-renewing cert without a person owning the renewal.

## Quick start

**Supported targets:** Debian/Ubuntu and the RHEL family (others may work but aren't tested), amd64 and arm64. One static binary and a systemd timer:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh
```

That pulls the matching release binary, checks its checksum, and runs the installer: it creates the `syscert` user and `/var/lib/syscert`, writes a starter config and secrets file, installs the systemd units, and enables the timer without starting it. Then you edit two files:

```sh
sudoedit /etc/syscert/syscert.toml      # subject, CA, challenge, distribute targets
sudoedit /etc/syscert/secrets           # e.g. CLOUDFLARE_DNS_API_TOKEN=...  (never in the .toml)

sudo -u syscert syscert dry-run --config-only                       # validate offline
sudo -u syscert syscert --staging --env-file /etc/syscert/secrets   # real run; --env-file loads creds
# happy? drop --staging, then: sudo systemctl start syscert.timer
```

For the full walkthrough, a complete minimal config and what each command prints back, see the [Quick start guide](docs/quick-start.md). If you'd rather read the script first, check the checksums yourself, build from source, or install by hand, [Advanced install](docs/advanced-install.md) covers all of it.

Uninstalling works the same way, no clone required: `curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall` (add `--purge` to take the certs, keys, and config with it). Details in [Advanced install](docs/advanced-install.md#uninstall).

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
| `syscert version` | Print the version and build info. |
| `syscert systemd-paths [--write]` | Print (or install, as root) the unit drop-in granting the sandboxed service write access to its distribute targets. |
| `syscert status` | Show config + the stored cert's dates (issued/expiry/renewal), account, and distribute targets. Offline. |

`--config` defaults to `/etc/syscert/syscert.toml`, or `$SYSCERT_CONFIG` if you set it. Secrets — DNS and CA tokens — come from the environment, never the TOML, and never reach the logs. The systemd service reads them from `/etc/syscert/secrets`. For a one-off manual run, point `--env-file` at that same file instead of exporting each variable by hand.

## Documentation

Full, canonical docs live in [`docs/`](docs/) and render on the website at **<https://syscert.tfindley.dev/docs/>**:

- [Quick start](docs/quick-start.md) — install → edit two files → done
- [Configuration reference](docs/configuration.md) — every `syscert.toml` option
- [Sample configurations](docs/examples.md) — a starter per CA + challenge, plus annotated [`examples/full.toml`](examples/full.toml)
- [Advanced install](docs/advanced-install.md) — verify checksums, build from source, manual systemd
- [Install with Ansible](docs/advanced-install/ansible.md) — fleet installs with the in-tree role
- [Distributing certs](docs/distributing.md) — artifacts, ownership/mode/SELinux, no reload hooks
- [Troubleshooting](docs/troubleshooting.md) · [FAQ](docs/faq.md) · [Roadmap](docs/roadmap.md) · [Changelog](CHANGELOG.md)

## Contributing

Commits follow [Conventional Commits](https://www.conventionalcommits.org/) and releases follow [Semantic Versioning](https://semver.org/) — see [RELEASING.md](RELEASING.md). The repo uses [pre-commit](https://pre-commit.com/); enable the hooks once with `pre-commit install` and `pre-commit install --hook-type commit-msg`.

## License

[AGPL-3.0](LICENSE).

## Author

[Tristan Findley](https://tfindley.co.uk). If you'd like to support the project: [☕ Ko-fi](https://ko-fi.com/tfindley).
