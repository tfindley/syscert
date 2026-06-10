// Command syscert obtains, renews, and distributes a hostname-based system TLS
// certificate. Bare `syscert` (the default) ensures the cert is present, fresh,
// and distributed — the command a systemd timer runs periodically.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// version is the build version, overridden at release time via
// -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches a subcommand and returns a process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return cmdEnsure(nil, stdout, stderr) // bare `syscert` → ensure
	}

	switch args[0] {
	case "-h", "--help", "help":
		usage(stdout)
		return 0
	case "version", "--version":
		fmt.Fprintf(stdout, "syscert %s\n", version)
		return 0
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "issue":
		return cmdIssue(rest, stdout, stderr)
	case "renew":
		return cmdRenew(rest, stdout, stderr)
	case "distribute":
		return cmdDistribute(rest, stdout, stderr)
	case "dry-run":
		return cmdDryRun(rest, stdout, stderr)
	case "trust":
		return cmdTrust(rest, stdout, stderr)
	case "void":
		return cmdVoid(rest, stdout, stderr)
	case "destroy":
		return cmdDestroy(rest, stdout, stderr)
	default:
		if strings.HasPrefix(cmd, "-") {
			return cmdEnsure(args, stdout, stderr) // bare flags → ensure with options
		}
		fmt.Fprintf(stderr, "syscert: unknown command %q\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `syscert — hostname-based system TLS certificate service

usage:
  syscert [--config <path>] [--staging]      ensure: issue/renew as needed, then
                                             distribute (the default; run by the timer)
  syscert issue   [--config <path>] [--staging]        obtain a fresh cert (no distribute)
  syscert renew   [--config <path>] [--staging] [--force]  renew if due (no distribute)
  syscert void    [--config <path>] [--staging] [--force]  revoke + reissue + distribute
  syscert distribute [--config <path>]       push stored artifacts to targets
  syscert dry-run [--config <path>] [--config-only]    validate + test ACME (nothing saved)
  syscert trust install [--config <path>] [--ca-file <path>]   add internal CA to the system trust store (root)
  syscert trust remove                       remove SysCert-managed CA anchors (root)
  syscert destroy [--config <path>] [--force]   wipe stored cert + ACME account (provider switch)
  syscert version

--config defaults to /etc/syscert/syscert.toml.
As a service, run bare 'syscert' from a systemd timer (the certbot model).
`)
}
