---
title: Compile from source
navLabel: Compile from source
description: Build the syscert static binary yourself with Go ≥ 1.26, then install it like any release binary.
order: 2
eyebrow: "// docs · advanced install · compile from source"
lede: Build the static binary yourself with Go, then install it like any other binary.
---

## Build from source

Requires **Go ≥ 1.26**. A local build derives its version from the checkout's VCS
info automatically (the tag, with a `+dirty` suffix when the tree has uncommitted
changes):

```sh
git clone https://github.com/tfindley/syscert.git
cd syscert
go build -o syscert ./cmd/syscert
./syscert version
```

## Install it

Once built, install the binary exactly like a downloaded one. Point the (idempotent)
installer at it for the full systemd setup:

```sh
sudo packaging/install.sh ./syscert
```

See [Manually → Install as a systemd service](/docs/advanced-install/manually/#install-as-a-systemd-service)
for what that creates, or run it [as a cron job](/docs/advanced-install/cron/) on an
appliance without systemd.

---

Next: [Manually](/docs/advanced-install/manually/) · [Configuration](/docs/configuration/) ·
[Quick start](/docs/quick-start/)
