# Example configurations

Drop one of these in at `/etc/syscert/syscert.toml` (or point `--config` / `SYSCERT_CONFIG` at it),
change `hostname` and `email`, and put any provider/CA credentials in the environment
(`/etc/syscert/secrets`) — never in the config. Provider variable names:
<https://go-acme.github.io/lego/dns/>.

| File | CA | Challenge | Notes |
|---|---|---|---|
| [`full.toml`](full.toml) | — | — | Every option, fully annotated (the reference). |
| [`letsencrypt-dns-01-gandiv5.toml`](letsencrypt-dns-01-gandiv5.toml) | Let's Encrypt | dns-01 | Public cert via Gandi LiveDNS; no inbound ports needed. |
| [`letsencrypt-http-01.toml`](letsencrypt-http-01.toml) | Let's Encrypt | http-01 | Simplest public setup; needs inbound :80. |
| [`letsencrypt-tls-alpn-01.toml`](letsencrypt-tls-alpn-01.toml) | Let's Encrypt | tls-alpn-01 | Modern :443-only challenge; no DNS, no :80. |
| [`vault-http-01.toml`](vault-http-01.toml) | HashiCorp Vault (`custom`) | http-01 | Internal CA; Vault reaches the host on :80. |
| [`vault-dns-01.toml`](vault-dns-01.toml) | HashiCorp Vault (`custom`) | dns-01 | Internal CA; role-scoped directory + EAB; no inbound ports. |
| [`stepca-dns-01.toml`](stepca-dns-01.toml) | Smallstep step-ca (`custom`) | dns-01 | Internal CA; step-ca supports all three challenges. |

**`ca` values:** `letsencrypt` (public — built-in URLs + `--staging`) or `custom` (any internal/
other ACME CA: Vault, step-ca, … via `directory_url`).

**Challenge support:** Let's Encrypt, Vault, and step-ca all support `dns-01` / `http-01` /
`tls-alpn-01`. (Vault's PKI ACME exposes `dns-01` in current versions — confirm yours.)

Validate any of these offline before deploying:

```sh
syscert dry-run --config-only --config examples/letsencrypt-dns-01-gandiv5.toml
```
