package main

import (
	"fmt"
	"io"
	"time"

	"github.com/tfindley/syscert/internal/distribute"
	"github.com/tfindley/syscert/internal/renewal"
	"github.com/tfindley/syscert/internal/store"
)

// cmdEnsure is the zero-argument default: obtain a cert if none exists, renew it
// if it's within the renewal window, then always distribute. This is the
// command a systemd timer runs periodically (idempotent).
func cmdEnsure(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("syscert", stderr)
	cfgPath := configFlag(fs)
	staging := fs.Bool("staging", false, "use the CA staging directory (Let's Encrypt) — for testing")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, subject, problems, err := loadAndCheck(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "syscert: load config: %v\n", err)
		return 2
	}
	if len(problems) > 0 {
		printProblems(stdout, problems)
		return 1
	}

	// Decide whether to obtain a (new or renewed) certificate.
	certPEM, readErr := store.ReadCurrentCert(cfg.Store.Path)
	obtain := false
	if readErr != nil {
		fmt.Fprintf(stdout, "no certificate in %s — issuing\n", cfg.Store.Path)
		obtain = true
	} else {
		due, err := renewal.Due(certPEM, cfg.Renewal.RenewBefore, time.Now())
		if err != nil {
			fmt.Fprintf(stderr, "syscert: %v\n", err)
			return 1
		}
		if due {
			fmt.Fprintln(stdout, "certificate due for renewal — renewing")
			obtain = true
		} else {
			fmt.Fprintln(stdout, "certificate not due for renewal")
		}
	}

	if obtain {
		noteConnectionTrust(cfg, stdout)
		if _, err := provision(cfg, subject, *staging); err != nil {
			fmt.Fprintf(stdout, "FAIL: %v\n", err)
			return 1
		}
	}

	// Always distribute, so targets stay in sync even when nothing was obtained.
	if err := distribute.New(cfg.Store.Path).Run(cfg.Distribute); err != nil {
		fmt.Fprintf(stderr, "syscert: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "OK: %s ensured; distributed to %d target(s)\n", subject, len(cfg.Distribute))
	return 0
}
