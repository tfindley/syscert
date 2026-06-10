package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/trust"
)

// cmdTrust manages the internal-CA chain in the system trust store (root-only).
func cmdTrust(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: syscert trust install|remove [--config <path>] [--ca-file <path>]")
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "install":
		return cmdTrustInstall(rest, stdout, stderr)
	case "remove":
		return cmdTrustRemove(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "syscert trust: unknown subcommand %q (want install|remove)\n", sub)
		return 2
	}
}

func cmdTrustInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("trust install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := configFlag(fs)
	caFile := fs.String("ca-file", "", "PEM of the CA to trust (overrides acme.ca_bundle)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(stderr, "trust install: must run as root")
		return 1
	}

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "trust install: load config: %v\n", err)
		return 2
	}

	source := *caFile
	if source == "" {
		source = cfg.ACME.CABundle
	}
	if source == "" {
		if config.IsPublicCA(cfg.ACME.CA) {
			fmt.Fprintln(stdout, "this is a public CA — the system already trusts it; nothing to do.")
			return 0
		}
		fmt.Fprintln(stderr, "trust install: no CA source — set acme.ca_bundle or pass --ca-file")
		return 2
	}

	caPEM, err := os.ReadFile(source)
	if err != nil {
		fmt.Fprintf(stderr, "trust install: read %s: %v\n", source, err)
		return 1
	}
	name, err := trust.CAName(caPEM)
	if err != nil {
		fmt.Fprintf(stderr, "trust install: %s: %v\n", source, err)
		return 1
	}
	mgr, err := trust.New()
	if err != nil {
		fmt.Fprintf(stderr, "trust install: %v\n", err)
		return 1
	}
	if err := mgr.Install(name, caPEM); err != nil {
		fmt.Fprintf(stderr, "trust install: %v\n", err)
		return 1
	}
	slog.Info("CA installed in system trust store", "name", name, "dir", mgr.AnchorDir)
	fmt.Fprintf(stdout, "OK: installed CA %q into the system trust store (%s)\n", name, mgr.AnchorDir)
	return 0
}

func cmdTrustRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("trust remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(stderr, "trust remove: must run as root")
		return 1
	}

	mgr, err := trust.New()
	if err != nil {
		fmt.Fprintf(stderr, "trust remove: %v\n", err)
		return 1
	}
	n, err := mgr.RemoveManaged()
	if err != nil {
		fmt.Fprintf(stderr, "trust remove: %v\n", err)
		return 1
	}
	slog.Info("CA anchors removed from system trust store", "count", n, "dir", mgr.AnchorDir)
	fmt.Fprintf(stdout, "OK: removed %d SysCert-managed CA anchor(s) from %s\n", n, mgr.AnchorDir)
	return 0
}
