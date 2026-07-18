package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// envInterval names the env var that sets --interval's default.
const envInterval = "SYSCERT_INTERVAL"

// minInterval is the floor for --interval: a self-scheduling loop must not hammer
// the CA. One-shot (the default, no --interval) is unaffected — it runs once.
const minInterval = time.Minute

// parseInterval turns the --interval value into a duration. An empty string means
// "one-shot" (return 0). A non-empty value is parsed with time.ParseDuration and
// rejected if below minInterval, so we never poll the CA more than once a minute.
func parseInterval(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --interval %q: %v", s, err)
	}
	if d < minInterval {
		return 0, fmt.Errorf("--interval %s is too short; the minimum is %s (avoids hammering the CA)", d, minInterval)
	}
	return d, nil
}

// sleeper sleeps for d unless ctx is cancelled first. It returns ctx.Err() if the
// context is cancelled during the wait, else nil. It is injectable so tests never
// really sleep. The production implementation is sleepCtx.
type sleeper func(ctx context.Context, d time.Duration) error

// sleepCtx is the production sleeper: a cancellable sleep built on a timer.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// runIntervalLoop runs cycle, sleeps for interval, and repeats until ctx is
// cancelled (SIGTERM/SIGINT), then returns exit code 0. It honours the
// signal-handling boundary in ADR-0046:
//   - a cancel that arrives mid-cycle lets the current cycle finish (cycle owns
//     ctx and decides whether to observe it), then the loop exits before sleeping;
//   - a cancel that arrives during the sleep exits without starting a new cycle.
//
// A cycle that returns an error is logged at error level and the loop continues
// to the next sleep — a sidecar must survive transient ACME/DNS/network blips.
// Fatal config/startup errors are handled by the caller before the loop starts.
func runIntervalLoop(ctx context.Context, interval time.Duration, cycle func(context.Context) error, sleep sleeper, stderr io.Writer) int {
	for {
		if err := cycle(ctx); err != nil {
			// Transient failure: log and keep scheduling. Never exit mid-loop.
			slog.Error("ensure cycle failed; will retry on the next interval", "err", err, "interval", interval.String())
			fmt.Fprintf(stderr, "syscert: ensure cycle failed (continuing): %v\n", err)
		}
		// If a signal arrived during the cycle, exit now rather than sleeping.
		if ctx.Err() != nil {
			return 0
		}
		if err := sleep(ctx, interval); err != nil {
			// Cancelled during the sleep — exit cleanly without a new cycle.
			return 0
		}
	}
}
