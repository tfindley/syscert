# Security

## Reporting a vulnerability

Report security vulnerabilities through the [GitHub Issues tracker](https://github.com/tfindley/syscert/issues). If a disclosure is sensitive, contact the maintainer directly using the profile linked from the README.

## Security assessment

The security assessment and risk register live at [docs/compliance/security.md](docs/compliance/security.md), rendered at <https://syscert.tfindley.dev/docs/compliance/security/>. Both are tool-backed, and every release re-runs them through `scripts/prerelease.sh` (`gosec`, `govulncheck`, and the tests).
