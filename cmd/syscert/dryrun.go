package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/tfindley/syscert/internal/acme"
	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/resolve"
	"github.com/tfindley/syscert/internal/validate"
)

// cmdDryRun validates the config and, unless --config-only, runs the full ACME
// order + challenge against the CA without persisting anything (like
// `certbot --dry-run`).
func cmdDryRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "", "path to syscert.toml")
	configOnly := fs.Bool("config-only", false, "validate config + resolve subject only; skip the ACME round-trip")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *cfgPath == "" {
		fmt.Fprintln(stderr, "dry-run: --config is required")
		return 2
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "dry-run: load config: %v\n", err)
		return 2
	}

	// --- Stage 1: config test (always) ---
	var problems []validate.Problem
	subject, err := resolve.FQDN(cfg.Cert.Hostname, nil)
	if err != nil {
		problems = append(problems, validate.Problem{Field: "cert.hostname", Message: err.Error()})
	}
	problems = append(problems, validate.Config(cfg)...)

	if len(problems) > 0 {
		fmt.Fprintf(stdout, "FAIL: %d config problem(s)\n", len(problems))
		for _, p := range problems {
			fmt.Fprintf(stdout, "  - %s: %s\n", p.Field, p.Message)
		}
		return 1
	}

	fmt.Fprintln(stdout, "config OK:")
	fmt.Fprintf(stdout, "  subject:   %s\n", subject)
	fmt.Fprintf(stdout, "  CA:        %s\n", cfg.ACME.CA)
	if eff := cfg.EffectiveChallenge(); eff != cfg.ACME.Challenge {
		fmt.Fprintf(stdout, "  challenge: %s (auto-selected for IP SANs; configured %s)\n", eff, cfg.ACME.Challenge)
	} else {
		fmt.Fprintf(stdout, "  challenge: %s\n", eff)
	}

	if *configOnly {
		return 0
	}

	// --- Stage 2: ACME dry-run (real order + challenge, nothing persisted) ---
	dir := acme.DirectoryURL(cfg, true)
	fmt.Fprintf(stdout, "\nACME dry-run against %s\n", dir)
	fmt.Fprintln(stdout, "(performs real challenge validation; no certificate is saved)")

	res, err := acme.DryRun(context.Background(), cfg, subject, acme.NewLegoObtainer())
	if err != nil {
		fmt.Fprintf(stdout, "FAIL: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "OK: dry-run issuance succeeded for %v (not saved)\n", res.Identifiers)
	return 0
}
