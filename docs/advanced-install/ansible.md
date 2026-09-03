---
title: Install with Ansible
navLabel: With Ansible
description: Fleet-install syscert with the packaging/ansible/syscert role — the Ansible-native equivalent of install.sh for many hosts at once, with a variable model that mirrors syscert.toml, vaulted secrets, and a download or local-binary install method for air-gapped fleets.
order: 5
eyebrow: "// docs · advanced install · ansible"
lede: One host at a time gets old fast. The syscert Ansible role does what install.sh does, natively, across a fleet — with variables that mirror syscert.toml and secrets you keep in ansible-vault.
---

Everything so far on this page's siblings installs **one host**. The `syscert` Ansible role does the same work — user, store, binary, config, secrets, systemd units, timer — for as many hosts as your inventory covers, in one run. It lives in-tree at [`packaging/ansible/`](https://github.com/tfindley/syscert/tree/main/packaging/ansible) and needs **ansible-core ≥ 2.21** (Ansible 14) plus one collection — `ansible.posix`, for the ACL that lets the service write into a privileged target directory. Install it with `ansible-galaxy collection install -r packaging/ansible/requirements.yml`; if you run the full `ansible` package rather than bare ansible-core, you already have it.

## Quick start

Point the role at a host group, give it the ACME account email and a CA, and hand it a DNS provider's credentials via `ansible-vault`:

```yaml
# playbook.yml
- hosts: web
  become: true
  roles:
    - role: syscert
      syscert_version: v0.4.0
      syscert_acme_email: tls@example.com
      syscert_acme_dns_provider: cloudflare
      syscert_secrets:
        CLOUDFLARE_DNS_API_TOKEN: "{{ vault_cloudflare_token }}"
```

```sh
ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass
```

That's a Let's Encrypt/dns-01 host, end to end. See [`packaging/ansible/playbook.example.yml`](https://github.com/tfindley/syscert/blob/main/packaging/ansible/playbook.example.yml) for a second worked example — Vault as a custom CA with EAB — and the role's own [`roles/syscert/README.md`](https://github.com/tfindley/syscert/blob/main/packaging/ansible/roles/syscert/README.md) for the full variable table.

## The variable model

Every role variable is prefixed `syscert_` and mirrors the `syscert.toml` schema 1:1 — `syscert_acme_ca` is `[acme] ca`, `syscert_cert_sans` is `[cert] sans`, `syscert_distribute` is the `[[distribute]]` array, and so on. Rather than duplicate that whole table here, read it alongside the [Configuration reference](/docs/configuration/): once you know one, you know the other. Required variables (the ACME email, a `directory_url` when `syscert_acme_ca` is `custom`, a DNS provider for a DNS challenge without IP SANs, and so on) have no default, so a missing or inconsistent set fails fast in `meta/argument_specs.yml` and `tasks/assert.yml`, before the role changes anything on the host.

What has **no TOML equivalent** — the install/system variables — is worth a table of its own:

| Variable | Default | Description |
|---|---|---|
| `syscert_install_method` | `download` | `download` (GitHub release, checksum-verified) or `local` (copy a binary from the controller). |
| `syscert_version` | *(required for `download`)* | Release tag to install, e.g. `v0.4.0`. Pin it — there's no implicit latest. |
| `syscert_local_binary` | *(required for `local`)* | Controller path to a pre-built binary. |
| `syscert_manage_distribute_dirs` | `true` | Create each writable directory the service needs, grant the syscert user on it with a POSIX ACL, and add it to the unit's `ReadWritePaths`. |
| `syscert_install_acl_package` | `true` | Install the `acl` package (provides `setfacl`). Minimal Debian, Ubuntu cloud and Raspberry Pi OS images do not ship it. |
| `syscert_bin_path` | `/usr/local/bin/syscert` | Install location on the host. |
| `syscert_user` / `syscert_group` | `syscert` | The service account the role creates and runs as. |
| `syscert_manage_user` | `true` | Whether the role creates the system user and group at all. |
| `syscert_state` | `present` | `present` installs/configures; `absent` uninstalls. |
| `syscert_purge` | `false` | With `absent`, also remove the store, config, secrets, and the service user. |

### Observability

Both outputs are off by default and map onto the [`[observe]`](/docs/configuration/#observe--metrics-and-inventory-facts) section. Enabling either one is enough for the role to create its directory, ACL-grant it, and add it to the unit's `ReadWritePaths` — the same treatment a privileged distribute target gets, because it is the same problem.

| Variable | Default | Description |
|---|---|---|
| `syscert_metrics_enabled` | `false` | Write a Prometheus node_exporter textfile after every run. |
| `syscert_metrics_file` | `/var/lib/node_exporter/textfile_collector/syscert.prom` | Where to write it. Must end `.prom`. |
| `syscert_ansible_facts_enabled` | `false` | Write an Ansible local-facts file after every run. |
| `syscert_ansible_facts_file` | `/etc/ansible/facts.d/syscert.fact` | Where to write it. Must end `.fact`; read back as `ansible_local.syscert`. |

The facts file is the reason this is worth turning on for a fleet: once it is in place, `ansible_local.syscert.not_after` reports expiry for every host in the inventory without syscert being involved.

## Secrets

DNS-provider credentials and the EAB HMAC go in `syscert_secrets` — an open-ended `KEY: value` map — and **never in the TOML**, same as everywhere else in syscert. Encrypt the values with `ansible-vault` (a vaulted `group_vars/*/vault.yml`, referenced with `{{ vault_… }}`, is the pattern both example playbooks use). The role renders `syscert_secrets` to `/etc/syscert/secrets` at `0640 root:syscert`, marks the task `no_log`, and never puts a secret value in a log or a `--diff`. See [Configuration](/docs/configuration/) for what belongs in `syscert_secrets` per CA and challenge, and `syscert_environment` for the non-secret sibling that renders `/etc/default/syscert`.

## Install methods

**`download`** (the default) fetches the pinned `syscert_version` release binary and verifies it against that release's `sha256sums.txt` before it is installed — the same checksum discipline as `packaging/install.sh`, and there is deliberately no variable to switch it off. The fetch happens on the **target host**, so it's the targets that need to reach GitHub releases, or an internal mirror via `syscert_download_base_url`. One caveat worth stating plainly: the binary and its checksum file come from the same origin, so this proves integrity, not provenance — point `syscert_download_base_url` only at an origin you trust.

**`local`** copies a binary you already built or downloaded onto the controller, via `syscert_local_binary`. This is the air-gapped route: no target host, and no controller, needs to reach the public internet for the binary itself. It's the fleet equivalent of the single-host [offline install](/docs/advanced-install/offline/) — pair it with an internal CA (Vault or step-ca) and neither the install nor the certificate lifecycle ever leaves your network.

## What it does on a host

For each host in scope, the role: creates the `syscert` system user and the `/var/lib/syscert` store; installs the (checksum-verified, for `download`) binary; renders `syscert.toml`, the secrets `EnvironmentFile`, and `/etc/default/syscert`; creates each writable directory it needs and grants the syscert user on it; installs the hardened `syscert.service` and `syscert.timer` units; and — before it ever enables the timer — **validates the rendered config** by running `syscert dry-run --config-only` as the `syscert` user. A host that fails that check fails the play, not a 3 a.m. renewal. Only once that passes does it enable (and, by default, start) the timer.

One of those steps is worth calling out, because it's the step people miss when they wire the units by hand. The unit runs under `ProtectSystem=strict`, which makes the entire filesystem read-only except the paths named in `ReadWritePaths` — so a certificate can be issued perfectly and then fail to *land*, with a read-only filesystem error, if a target's directory isn't granted. A static unit can't know your targets; the role does, so it derives that list from your `syscert_distribute` targets **and** any enabled `[observe]` output, creates the directories, and grants the syscert user on each with a POSIX ACL — because being *allowed* by the sandbox is only half of it, and a root-owned `0755` directory would still refuse a non-root service. Set `syscert_manage_distribute_dirs: false` if you manage those directories elsewhere — they still have to exist, or the unit won't start.

## Uninstall

```yaml
- role: syscert
  syscert_state: absent      # remove the units + binary; keep the store, config, and user
  # syscert_purge: true      # also remove the store, config, secrets, and the user
```

`syscert_purge: true` is destructive: it deletes `/var/lib/syscert` (the ACME account key and every certificate the host holds) along with the config and the service user. There's no confirmation prompt at the Ansible layer the way `install.sh --uninstall --purge` has one at the terminal — the playbook run itself is the confirmation, so keep it out of a play you run unattended against a group you didn't mean to purge.

## Tags

Run a subset of the role with `--tags`: `syscert_assert`, `syscert_user`, `syscert_binary`, `syscert_config`, `syscert_secrets`, `syscert_distribute`, `syscert_service`, `syscert_uninstall`. Useful for, say, re-rendering just the config and secrets (`--tags syscert_config,syscert_secrets`) after a `syscert_distribute` change, without touching the installed binary or the timer state.

## Testing the role itself

The role ships an [ansible-lint](https://ansible.readthedocs.io/projects/lint/) config pinned to the `production` profile and a [Molecule](https://ansible.readthedocs.io/projects/molecule/) scenario under `roles/syscert/molecule/` (needs Podman or Docker and a systemd-capable image). See [`packaging/ansible/README.md`](https://github.com/tfindley/syscert/blob/main/packaging/ansible/README.md) for the exact commands.

---

Next: [Configuration](/docs/configuration/) · [Install offline (air-gapped)](/docs/advanced-install/offline/) · [Install manually](/docs/advanced-install/manually/)
