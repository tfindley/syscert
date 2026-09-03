# SysCert — Ansible

Fleet installs of [SysCert](https://github.com/tfindley/syscert) — the Ansible-native
equivalent of `packaging/install.sh`. Configure many hosts at once: each gets the `syscert`
user + store, the checksum-verified binary, a rendered `syscert.toml` + secrets file, and the
systemd service + timer (enabled and started by default).

## Layout

```
packaging/ansible/
  ansible.cfg            # roles_path = roles
  .ansible-lint          # profile: production
  requirements.yml       # no external collections (ansible.builtin only)
  playbook.example.yml   # Let's Encrypt + Vault/EAB examples
  roles/syscert/         # the role (see roles/syscert/README.md for all variables)
```

## Quick start

```sh
cd packaging/ansible
# 1. Inventory: a [public_web] / [internal] host or your own groups.
# 2. Put credentials in an ansible-vault file referenced by the playbook.
ansible-playbook -i inventory.ini playbook.example.yml --ask-vault-pass
```

Minimal play:

```yaml
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

## Requirements

- **ansible-core ≥ 2.21** (Ansible 14); the role uses only `ansible.builtin`.
- Targets EL 9/10, Debian 12, Ubuntu 22.04/24.04.
- Controller reaches the GitHub releases (default `download`), or set
  `syscert_install_method: local` with `syscert_local_binary`.

## Variables & secrets

Full reference: [`roles/syscert/README.md`](roles/syscert/README.md) and the annotated
[`roles/syscert/defaults/main.yml`](roles/syscert/defaults/main.yml). Every variable is
prefixed `syscert_` and mirrors the `syscert.toml` schema. Put DNS/CA credentials and the EAB
HMAC in `syscert_secrets` and encrypt them with `ansible-vault`; the role writes them
`0640 root:syscert` and never logs them.

## Validation & testing

```sh
ansible-lint                                   # production profile, must be clean
ansible-playbook --syntax-check playbook.example.yml
cd roles/syscert && molecule test              # needs Podman/Docker (systemd images)
```

The role also validates the rendered `syscert.toml` on each run via
`syscert dry-run --config-only`, and `meta/argument_specs.yml` + `tasks/assert.yml` reject bad or
incomplete variable sets before anything is changed.

## Uninstall

```yaml
- role: syscert
  syscert_state: absent      # remove units + binary; keep data
  # syscert_purge: true      # also remove the store, config, secrets, and the user
```
