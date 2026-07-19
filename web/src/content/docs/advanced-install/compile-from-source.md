---
title: Compile from source
navLabel: Compile from source
description: Build the syscert static binary yourself with Go ≥ 1.26, then install it like any release binary.
order: 2
eyebrow: "// docs · advanced install · compile from source"
lede: Build the static binary yourself with Go, then install it like any other binary.
---

## Build from source

You'll need **Go ≥ 1.26**. A local build picks up its version from the checkout's VCS info on its own: the tag, plus a `+dirty` suffix when the tree has uncommitted changes.

```sh
git clone https://github.com/tfindley/syscert.git
cd syscert
go build -o syscert ./cmd/syscert
./syscert version
```

## Install it

Once it's built, install it exactly like a downloaded binary. Point the installer at it (it's idempotent) and you get the whole systemd setup:

```sh
sudo packaging/install.sh ./syscert
```

See [Manually → Install as a systemd service](/docs/advanced-install/manually/#install-as-a-systemd-service) for what that sets up. On an appliance with no systemd, run it [as a cron job](/docs/advanced-install/cron/) instead.

---

Next: [Manually](/docs/advanced-install/manually/) · [Configuration](/docs/configuration/) · [Quick start](/docs/quick-start/)
