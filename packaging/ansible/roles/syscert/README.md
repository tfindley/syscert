# Ansible role: `syscert`

Installs and configures [SysCert](https://github.com/tfindley/syscert) — a least-privilege
systemd service that gives a host its own auto-renewing TLS certificate (Let's Encrypt,
HashiCorp Vault, or Smallstep step-ca) and distributes it to local consumers. This role does
natively, across a fleet, what `packaging/install.sh` does on one host: creates the `syscert`
system user and store, installs the (checksum-verified) binary, renders `syscert.toml`, the
secrets `EnvironmentFile`, and the systemd service + timer, then enables (and by default starts)
the timer.

## Requirements

- **ansible-core ≥ 2.21** (Ansible 14). The role uses only `ansible.builtin`.
- Targets: EL 9/10, Debian 12, Ubuntu 22.04/24.04 (systemd hosts).
- The controller reaches the GitHub releases (default `download` method) — or supply a binary
  with the `local` method.

## Role variables

Every variable is prefixed `syscert_` and mirrors the `syscert.toml` schema. Required variables
have **no default** (enforced by `meta/argument_specs.yml`); conditional requirements are enforced
by `tasks/assert.yml`. Full annotated defaults: [`defaults/main.yml`](defaults/main.yml).

### Install / system

| Variable | Default | Description |
|---|---|---|
| `syscert_install_method` | `download` | `download` (GitHub release, checksum-verified) or `local` (copy from controller). |
| `syscert_version` | _(required for download)_ | Release tag, e.g. `v0.4.0`. Pin it. |
| `syscert_local_binary` | _(required for local)_ | Controller path to a pre-built binary. |
| `syscert_download_base_url` | GitHub releases URL | Override for an internal mirror. |
| `syscert_manage_distribute_dirs` | `true` | Create each distribute target's parent directory and grant it in the unit's `ReadWritePaths`. |
| `syscert_bin_path` | `/usr/local/bin/syscert` | Install location. |
| `syscert_user` / `syscert_group` | `syscert` | Service account. |
| `syscert_manage_user` | `true` | Create the system user/group. |
| `syscert_config_dir` | `/etc/syscert` | Config directory. |
| `syscert_state` | `present` | `present` or `absent`. |
| `syscert_purge` | `false` | With `absent`, also remove store/config/secrets/user. |

### Certificate, ACME, store (mirror the TOML)

| Variable | Default | TOML key |
|---|---|---|
| `syscert_cert_hostname` | `""` (→ FQDN) | `[cert] hostname` |
| `syscert_cert_sans` | `[]` | `[cert] sans` |
| `syscert_cert_ip_sans` | `[]` | `[cert] ip_sans` |
| `syscert_cert_key_type` | `ec256` | `[cert] key_type` |
| `syscert_cert_reuse_key` | `false` | `[cert] reuse_key` — accepted but **not yet applied** by syscert; every renewal still generates a fresh keypair. |
| `syscert_acme_ca` | `letsencrypt` | `[acme] ca` |
| `syscert_acme_email` | _(required)_ | `[acme] email` |
| `syscert_acme_directory_url` | `""` (required when `ca=custom`) | `[acme] directory_url` |
| `syscert_acme_challenge` | `dns-01` | `[acme] challenge` |
| `syscert_acme_profile` | `""` | `[acme] profile` |
| `syscert_acme_ca_bundle` | `""` | `[acme] ca_bundle` |
| `syscert_acme_dns_provider` | `""` (required for a DNS challenge) | `[acme.dns] provider` |
| `syscert_acme_dns_propagation_check` | `""` | `[acme.dns] propagation_check` |
| `syscert_acme_eab_kid` | `""` (requires `SYSCERT_EAB_HMAC`) | `[acme.eab] kid` |
| `syscert_store_path` | `/var/lib/syscert` | `[store] path` |
| `syscert_store_group` | `""` | `[store] group` |
| `syscert_store_dir_mode` | `"0700"` | `[store] dir_mode` |
| `syscert_store_archive_keep` | `0` | `[store] archive_keep` |
| `syscert_bundle_order` | `[cert, chain, root, key]` | `[bundle] order` |
| `syscert_distribute` | `[]` | `[[distribute]]` (list of `{artifact, path, owner, group, mode, selinux_context}`) |
| `syscert_renewal_renew_before` | `""` | `[renewal] renew_before` |
| `syscert_logging_level` | `info` | `[logging] level` |
| `syscert_logging_format` | `text` | `[logging] format` |

### Secrets, environment, timer

| Variable | Default | Description |
|---|---|---|
| `syscert_secrets` | `{}` | Open-ended `KEY: value` → `/etc/syscert/secrets` (`0640`, never logged). DNS-provider creds + `SYSCERT_EAB_HMAC`. |
| `syscert_environment` | `{}` | Open-ended `KEY: value` → `/etc/default/syscert` (non-secret). |
| `syscert_timer_enabled` | `true` | Enable the timer. |
| `syscert_timer_state` | `started` | `started` or `stopped`. |
| `syscert_timer_on_calendar` / `_on_boot_sec` / `_randomized_delay_sec` / `_persistent` | `daily` / `5min` / `12h` / `true` | Timer schedule. |
| `syscert_service_exec_extra_args` | `""` | Appended to `ExecStart` (e.g. `--staging`). |
| `syscert_service_ambient_capabilities` | `""` (auto) | Override the auto `CAP_CHOWN` [+ `CAP_NET_BIND_SERVICE`]. |

## Secrets

Put DNS-provider credentials and the EAB HMAC in `syscert_secrets` and **encrypt them with
`ansible-vault`** — the role writes them `0640 root:syscert` and never logs them (`no_log`). The
config (`syscert.toml`) is non-secret but still rendered `0640` (it carries the internal
`directory_url`, email, and EAB `kid`).

## Example

```yaml
- name: Issue and distribute the host certificate
  hosts: web
  become: true
  roles:
    - role: syscert
      syscert_version: v0.4.0
      syscert_acme_email: tls@example.com
      syscert_acme_dns_provider: cloudflare
      syscert_secrets:
        CLOUDFLARE_DNS_API_TOKEN: "{{ vault_cloudflare_token }}"
      syscert_distribute:
        - { artifact: fullchain, path: /etc/nginx/tls/fullchain.pem, owner: root, group: root, mode: "0644" }
        - { artifact: privkey,   path: /etc/nginx/tls/privkey.pem,   owner: root, group: root, mode: "0600" }
```

See [`../playbook.example.yml`](../playbook.example.yml) for Let's Encrypt and Vault/EAB examples,
and [`molecule/`](molecule/) for the test scenario.

## License

MIT.
