package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/tfindley/syscert/internal/acme"
	"github.com/tfindley/syscert/internal/distribute"
	"github.com/tfindley/syscert/internal/store"
)

// cmdVoid revokes the current certificate, then reissues and distributes a fresh
// one. Interactive by default; --force skips the confirmation.
func cmdVoid(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("void", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", defaultConfigPath, "path to syscert.toml")
	staging := fs.Bool("staging", false, "use the CA staging directory (Let's Encrypt) — for testing")
	force := fs.Bool("force", false, "skip the interactive confirmation")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, subject, problems, err := loadAndCheck(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "void: load config: %v\n", err)
		return 2
	}
	if len(problems) > 0 {
		printProblems(stdout, problems)
		return 1
	}

	certPEM, err := store.ReadCurrentCert(cfg.Store.Path)
	if err != nil {
		fmt.Fprintf(stderr, "void: no current certificate in %s (nothing to void): %v\n", cfg.Store.Path, err)
		return 1
	}

	prompt := fmt.Sprintf("Revoke and reissue %s? The current certificate will be invalidated. [y/N] ", subject)
	if !confirm(stdout, os.Stdin, prompt, *force) {
		fmt.Fprintln(stdout, "aborted.")
		return 0
	}

	noteConnectionTrust(cfg, stdout)

	revokeFailed := false
	if err := acme.RevokeCert(context.Background(), cfg, certPEM, acme.NewLegoObtainer(), *staging); err != nil {
		fmt.Fprintf(stderr, "warning: revocation failed (the old certificate was NOT revoked): %v\n", err)
		revokeFailed = true
	} else {
		slog.Info("certificate revoked", "subject", subject)
		fmt.Fprintln(stdout, "revoked the current certificate")
	}

	if _, err := provision(cfg, subject, *staging); err != nil {
		fmt.Fprintf(stdout, "FAIL: reissue: %v\n", err)
		return 1
	}
	if err := distribute.New(cfg.Store.Path).Run(cfg.Distribute); err != nil {
		fmt.Fprintf(stderr, "void: distribute: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "OK: reissued and distributed %s\n", subject)
	if revokeFailed {
		return 1 // partial: reissued, but revocation did not happen
	}
	return 0
}
