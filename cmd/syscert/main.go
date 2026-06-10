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

Obtains and auto-renews a TLS certificate for this host (Let's Encrypt, Vault,
or step-ca) and distributes it to local consumers. As a service it's just bare
'syscert' run by a systemd timer — no daemon.

Usage:
  syscert [flags]              ensure the cert is issued, renewed, and distributed
                               (the default — this is what the timer runs)
  syscert <command> [flags]

Commands:
  issue        Obtain a fresh certificate into the store (no distribute)
  renew        Renew only if due, into the store (no distribute)
  distribute   Copy the stored artifacts to the configured targets
  dry-run      Validate config and test the ACME flow; nothing is saved
  void         Revoke the current certificate, then reissue and distribute
  destroy      Wipe the stored cert + ACME account (e.g. when switching CA)
  trust        Add/remove the internal CA in the system trust store (root)
  version      Print the version

Common flags:
  --config <path>   Config file. Default /etc/syscert/syscert.toml; or set
                    SYSCERT_CONFIG. An explicit --config wins over the env var.
  --staging         Use the CA's staging environment (issue/renew/void/ensure).
  --force           Skip the interactive confirmation (renew/void/destroy).
  --config-only     dry-run only: check config, skip the live ACME test.

Credentials (DNS-provider / CA tokens) come from the environment, never the
config file — the service loads them from /etc/syscert/secrets. Look up the
variables your provider needs at: https://go-acme.github.io/lego/dns/

Run 'syscert <command> --help' for a command's own flags.
Docs: https://github.com/tfindley/syscert
`)
}
