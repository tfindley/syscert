// Command syscert obtains, renews, and distributes a hostname-based system TLS
// certificate. Bare `syscert` (the default) ensures the cert is present, fresh,
// and distributed — the command a systemd timer runs periodically.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
)

// version and date are stamped at release time via
//
//	-ldflags "-X main.version=v1.2.3 -X main.date=2026-06-10T12:00:00Z"
//
// For non-release builds they stay at their defaults and are enriched from the
// VCS info Go embeds in the binary (see buildInfo).
var (
	version = "dev"
	date    = ""
)

// repoURL is shown by `syscert version`.
const repoURL = "https://github.com/tfindley/syscert"

// buildInfo resolves the version and build date to report. A release build uses
// the ldflags-stamped values; otherwise it falls back to what `go build` /
// `go install` embed — the module version (for `go install …@vX.Y.Z`), else the
// VCS commit + dirty flag, and the commit time as the build date.
func buildInfo() (ver, built string) {
	ver, built = version, date
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver, built
	}

	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		case "vcs.time":
			if built == "" {
				built = s.Value // commit time, as a build-date fallback
			}
		}
	}

	if ver == "dev" {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			ver = v // e.g. go install …@vX.Y.Z, or a tag-derived pseudo-version
		} else if rev != "" {
			if len(rev) > 12 {
				rev = rev[:12]
			}
			ver = "dev+" + rev
			if dirty {
				ver += "-dirty"
			}
		}
	}
	return ver, built
}

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
		ver, built := buildInfo()
		fmt.Fprintf(stdout, "syscert %s\n", ver)
		if built != "" {
			fmt.Fprintf(stdout, "built   %s\n", built)
		}
		fmt.Fprintf(stdout, "%s\n", repoURL)
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
	case "status":
		return cmdStatus(rest, stdout, stderr)
	case "systemd-paths":
		return cmdSystemdPaths(rest, stdout, stderr)
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
	fmt.Fprintf(w, `syscert — hostname-based system TLS certificate service

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
  destroy      Wipe the stored cert + ACME account (--keep-account drops only the cert)
  status       Show config + stored cert (issued/expiry/renewal), account, targets
  trust        Add/remove the internal CA in the system trust store (root)
  systemd-paths Print (or --write, as root) the unit drop-in granting the
               sandboxed service write access to its distribute targets
  version      Print the version

Common flags:
  --config <path>   Config file. Default /etc/syscert/syscert.toml; or set
                    SYSCERT_CONFIG. An explicit --config wins over the env var.
  --env-file <path> Load DNS/CA credentials from a systemd EnvironmentFile before
                    issuing (repeatable; the environment wins). For manual runs.
  --staging         Use the CA's staging environment (issue/renew/void/ensure).
  --interval <dur>  Bare syscert only: loop, sleeping this long between cycles,
                    until SIGTERM/SIGINT (e.g. 12h; min 1m; env SYSCERT_INTERVAL).
                    For containers/appliances with no systemd timer.
  --force           void/destroy: skip the interactive confirmation.
                    renew: renew even if the certificate is not yet due.
  --config-only     dry-run only: check config, skip the live ACME test.

Credentials (DNS-provider / CA tokens) come from the environment, never the
config file — the service loads them from /etc/syscert/secrets. For a manual run,
point --env-file at that same file instead of exporting each variable. Look up the
variables your provider needs at: https://go-acme.github.io/lego/dns/

Run 'syscert <command> --help' for a command's own flags.
Docs: %s
`, repoURL)
}
