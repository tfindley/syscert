---
title: Advanced install
navLabel: Advanced install
description: Install syscert your way — a verified release binary with manual systemd setup, a build from source, or a cron job on an appliance without systemd — plus uninstall.
order: 2
eyebrow: "// docs · advanced install"
lede: The one-liner is the fast path. These are the verify-every-byte routes — download + verify, build from source, run from cron on an appliance, and uninstall.
---

Supported targets are Debian/Ubuntu and the RHEL family, on amd64 and arm64. Others might work, but they're untested. The [install page](/install/) covers the one-line installer and an inspect-first walkthrough; the pages below are the building blocks it automates, so pick the route that fits.

## Pick an install route

- **[Manually](/docs/advanced-install/manually/)** — download a verified release binary and run the systemd install by hand. The closest equivalent to the one-liner, step by step.
- **[Compile from source](/docs/advanced-install/compile-from-source/)** — build the binary yourself with Go, then install it.
- **[As a cron job](/docs/advanced-install/cron/)** — for appliances and NAS boxes **without systemd** (e.g. Asustor ADM): schedule `syscert` from cron instead of the systemd timer.

If syscert's already installed, [Upgrading](/docs/advanced-install/upgrading/) covers the in-place binary swap that keeps your config and certificates.

## Uninstall

If you installed with the one-liner, remove it the same way. No clone needed:

```sh
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall          # keep data
curl -fsSL https://syscert.tfindley.dev/install.sh | sudo sh -s -- --uninstall --purge  # + data, config, user
```

From a source checkout, run the script directly:

```sh
sudo packaging/install.sh --uninstall            # remove units + binary, keep data
sudo packaging/install.sh --uninstall --purge    # also remove /var/lib/syscert, /etc/syscert, user
```

`--purge` can't be undone, so it asks you to confirm at the terminal; set `SYSCERT_ASSUME_YES=1` to skip that prompt (handy in automation). If you ran syscert from cron on an appliance, there's nothing system-level to remove. See [As a cron job](/docs/advanced-install/cron/).

---

Next: [Quick start](/docs/quick-start/) · [Configuration](/docs/configuration/) · [Sample configurations](/docs/examples/)
