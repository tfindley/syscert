package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/distribute"
	"github.com/tfindley/syscert/internal/renewal"
	"github.com/tfindley/syscert/internal/store"
)

// cmdEnsure is the zero-argument default: obtain a cert if none exists, renew it
// if it's within the renewal window, then always distribute. This is the command
// a systemd timer runs periodically (idempotent).
//
// With --interval set, it instead self-schedules: it validates once up front,
// then runs the ensure cycle on a loop, sleeping for the interval between runs,
// until SIGTERM/SIGINT — a scheduler for non-systemd contexts (containers/
// appliances) per ADR-0046. No --interval keeps the one-shot default unchanged.
func cmdEnsure(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("syscert", stderr)
	cfgPath := configFlag(fs)
	envPaths := envFileFlag(fs)
	staging := fs.Bool("staging", false, "use the CA staging directory (Let's Encrypt) — for testing")
	intervalDef := os.Getenv(envInterval)
	interval := fs.String("interval", intervalDef,
		"run on a loop, sleeping this long between cycles, until SIGTERM/SIGINT (e.g. 12h; min 1m; env: SYSCERT_INTERVAL). Default: run once and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if code := loadEnvFiles(*envPaths, stderr); code != 0 {
		return code
	}

	// Validate once, up front: a fatal config/interval problem must exit non-zero
	// BEFORE any loop starts (it must not be swallowed as a per-cycle error).
	every, err := parseInterval(*interval)
	if err != nil {
		fmt.Fprintf(stderr, "syscert: %v\n", err)
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
	if !storeAccessGuard("syscert", cfg.Store.Path, stderr) {
		return 1
	}
	// Report unwritable targets before any network work, so the cause is at the
	// top of the journal rather than buried behind an errno at the end. This does
	// not stop the run: see warnDistributeTargets.
	warnDistributeTargets("syscert", cfg, stderr)

	cycle := func(ctx context.Context) error {
		return ensureCycle(ctx, cfg, subject, *staging, stdout, stderr)
	}

	// One-shot (default): run a single cycle and translate its error to an exit code.
	if every == 0 {
		if err := cycle(context.Background()); err != nil {
			fmt.Fprintf(stdout, "FAIL: %v\n", err)
			return 1
		}
		return 0
	}

	// Interval mode: loop until a stop signal cancels the context, finishing the
	// current cycle first (ADR-0046).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	fmt.Fprintf(stdout, "running on a %s interval (SIGTERM/SIGINT to stop)\n", every)
	return runIntervalLoop(ctx, every, cycle, sleepCtx, stderr)
}

// ensureCycle runs one idempotent ensure pass: decide whether a (new or renewed)
// certificate is needed, obtain + persist it if so, then always distribute so
// targets stay in sync. It is identical whether run one-shot or per loop
// iteration, and returns an error rather than an exit code so the caller decides
// how to react (one-shot → exit 1; loop → log and continue). ctx flows into the
// obtain step but the cycle is never abandoned half-way (it owns the store write).
func ensureCycle(ctx context.Context, cfg *config.Config, subject string, staging bool, stdout, stderr io.Writer) error {
	certPEM, readErr := store.ReadCurrentCert(cfg.Store.Path)
	obtain := false
	if readErr != nil {
		fmt.Fprintf(stdout, "no certificate in %s — issuing\n", cfg.Store.Path)
		obtain = true
	} else {
		due, err := renewal.Due(certPEM, cfg.Renewal.RenewBefore, time.Now())
		if err != nil {
			return err
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
		if _, err := provision(cfg, subject, staging); err != nil {
			return err
		}
	}

	// Always distribute, so targets stay in sync even when nothing was obtained.
	distErr := distribute.New(cfg.Store.Path).Run(cfg.Distribute)

	// Written before the error is returned, and regardless of it: a delivery that
	// failed is precisely the state monitoring and inventory need to see. No-op
	// unless [observe] configures an output.
	writeObservations(cfg, subject, time.Now(), stderr)

	if distErr != nil {
		return distErr
	}
	fmt.Fprintf(stdout, "OK: %s ensured; distributed to %d target(s)\n", subject, len(cfg.Distribute))
	if s := certLine(cfg, time.Now()); s != "" { // surfaces in `systemctl status syscert`
		fmt.Fprintf(stdout, "%s\n", s)
	}
	return nil
}
