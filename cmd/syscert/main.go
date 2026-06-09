// Command syscert obtains and renews a hostname-based system TLS certificate.
//
// This is the walking skeleton: `dry-run` is functional; the other verbs are
// stubs to be filled in as the build progresses (see docs/user-flows.md).
package main

import (
	"fmt"
	"io"
	"os"
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
		usage(stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "dry-run":
		return cmdDryRun(rest, stdout, stderr)
	case "run", "issue", "renew", "void", "destroy", "trust":
		fmt.Fprintf(stderr, "syscert %s: not implemented yet\n", cmd)
		return 1
	case "version", "--version":
		fmt.Fprintf(stdout, "syscert %s\n", version)
		return 0
	case "-h", "--help", "help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "syscert: unknown command %q\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `syscert — hostname-based system TLS certificate service

usage:
  syscert dry-run --config <path>   config test + ACME dry-run (no cert saved)
  syscert dry-run --config <path> --config-only   validate config only (offline)
  syscert run     --config <path>   issuance + renewal loop            [todo]
  syscert issue   --config <path>   one-shot issuance                  [todo]
  syscert renew   --config <path>   force a renewal                    [todo]
  syscert void    --config <path>   revoke/discard + reissue           [todo]
  syscert destroy --config <path>   tear down + re-provision           [todo]
  syscert trust install|remove      manage system trust store (root)   [todo]
`)
}
