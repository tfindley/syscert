package main

import (
	"fmt"
	"io"
	"time"

	"github.com/tfindley/syscert/internal/renewal"
	"github.com/tfindley/syscert/internal/store"
)

// cmdRenew renews the certificate only if it is within its renewal window (or
// --force). Like issue, it writes to the store but does NOT distribute.
func cmdRenew(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("renew", stderr)
	cfgPath := configFlag(fs)
	envPaths := envFileFlag(fs)
	staging := fs.Bool("staging", false, "use the CA staging directory (Let's Encrypt) — for testing")
	force := fs.Bool("force", false, "renew even if the current cert is not yet due")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if code := loadEnvFiles(*envPaths, stderr); code != 0 {
		return code
	}

	cfg, subject, problems, err := loadAndCheck(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "renew: load config: %v\n", err)
		return 2
	}
	if len(problems) > 0 {
		printProblems(stdout, problems)
		return 1
	}

	certPEM, err := store.ReadCurrentCert(cfg.Store.Path)
	if err != nil {
		fmt.Fprintf(stderr, "renew: no current certificate in %s (run `syscert issue` first): %v\n", cfg.Store.Path, err)
		return 1
	}

	if !*force {
		due, err := renewal.Due(certPEM, cfg.Renewal.RenewBefore, time.Now())
		if err != nil {
			fmt.Fprintf(stderr, "renew: %v\n", err)
			return 1
		}
		if !due {
			fmt.Fprintln(stdout, "not due for renewal (use --force to renew anyway)")
			return 0
		}
	}

	noteConnectionTrust(cfg, stdout)
	if _, err := provision(cfg, subject, *staging); err != nil {
		fmt.Fprintf(stdout, "FAIL: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "OK: renewed %s in %s (not distributed — run `syscert distribute`)\n", subject, cfg.Store.Path)
	return 0
}
