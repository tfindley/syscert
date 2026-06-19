---
title: Advanced install
navLabel: Advanced install
description: Install syscert your way — a verified release binary with manual systemd setup, a build from source, or a cron job on an appliance without systemd — plus uninstall.
order: 2
eyebrow: "// docs · advanced install"
lede: The one-liner is the fast path. These are the verify-every-byte routes — download + verify, build from source, run from cron on an appliance, and uninstall.
---

**Supported targets:** Debian/Ubuntu and the RHEL family (others may work but
aren't tested), on amd64 and arm64. For the one-line installer and an
inspect-first walkthrough, see the [install page](/install/); the pages below are
the building blocks it automates — pick the route that fits.

## Pick an install route

- **[Manually](/docs/advanced-install/manually/)** — download a verified release
  binary and run the systemd install by hand. The closest equivalent to the
  one-liner, step by step.
- **[Compile from source](/docs/advanced-install/compile-from-source/)** — build the
  binary yourself with Go, then install it.
- **[As a cron job](/docs/advanced-install/cron/)** — for appliances and NAS boxes
  **without systemd** (e.g. Asustor ADM): schedule `syscert` from cron instead of the
  systemd timer.

**Already installed?** See [Upgrading](/docs/advanced-install/upgrading/) — an in-place binary swap
that preserves your config and certificates.

## Uninstall

Installed with the one-liner? Remove it the same way — no clone needed:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall          # keep data
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge  # + data, config, user
```

From a source checkout, use the script directly:

```sh
sudo packaging/install.sh --uninstall            # remove units + binary, keep data
sudo packaging/install.sh --uninstall --purge    # also remove /var/lib/syscert, /etc/syscert, user
```

`--purge` is irreversible, so it asks you to confirm on the terminal; set
`SYSCERT_ASSUME_YES=1` to skip the prompt (e.g. in automation). Ran it from cron on an
appliance instead? There's nothing system-level to remove — see
[As a cron job](/docs/advanced-install/cron/).

---

Next: [Quick start](/docs/quick-start/) · [Configuration](/docs/configuration/) ·
[Sample configurations](/docs/examples/)
