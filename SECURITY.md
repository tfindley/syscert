# Security

## Reporting a vulnerability

Please report security vulnerabilities via the [GitHub Issues tracker](https://github.com/tfindley/syscert/issues). For sensitive disclosures, contact the maintainer directly via the profile linked in the README.

## Security assessment

A published, tool-backed security assessment and risk register is maintained at
[docs/compliance/security.md](docs/compliance/security.md) (rendered at <https://syscert.tfindley.dev/docs/compliance/security/>).
It is re-run on every release via `scripts/prerelease.sh` (`gosec` + `govulncheck` + tests).
