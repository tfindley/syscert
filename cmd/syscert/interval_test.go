package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestParseIntervalRejectsSubMinute pins the < 1m guard: a too-short interval is
// a fatal startup error (so we never hammer the CA), surfaced with a clear message.
func TestParseIntervalRejectsSubMinute(t *testing.T) {
	for _, in := range []string{"30s", "1s", "59s"} {
		if _, err := parseInterval(in); err == nil {
			t.Errorf("parseInterval(%q): want error, got nil", in)
		} else if !strings.Contains(err.Error(), "1m") {
			t.Errorf("parseInterval(%q): error should mention the 1m minimum, got %q", in, err)
		}
	}
}

// TestParseIntervalAcceptsValid covers the empty (one-shot) case and a valid
// duration at/above the minimum.
func TestParseIntervalAcceptsValid(t *testing.T) {
	d, err := parseInterval("")
	if err != nil {
		t.Fatalf("parseInterval(\"\"): unexpected error %v", err)
	}
	if d != 0 {
		t.Errorf("parseInterval(\"\"): want 0 (one-shot), got %v", d)
	}
	d, err = parseInterval("12h")
	if err != nil {
		t.Fatalf("parseInterval(\"12h\"): unexpected error %v", err)
	}
	if d != 12*time.Hour {
		t.Errorf("parseInterval(\"12h\"): got %v, want 12h", d)
	}
	if _, err := parseInterval("1m"); err != nil {
		t.Errorf("parseInterval(\"1m\"): want accepted at the minimum, got %v", err)
	}
	if _, err := parseInterval("not-a-duration"); err == nil {
		t.Errorf("parseInterval(garbage): want a parse error")
	}
}

// recordingSleeper is an injectable sleeper that records each requested duration
// and lets a test control how many sleeps happen before the context is cancelled.
type recordingSleeper struct {
	mu      sync.Mutex
	calls   []time.Duration
	cancel  context.CancelFunc
	max     int // cancel the loop's context once this many sleeps have been requested
	failNth int // if > 0, return ctx.Err() on the failNth sleep (simulates cancel during sleep)
}

func (s *recordingSleeper) sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	s.calls = append(s.calls, d)
	n := len(s.calls)
	s.mu.Unlock()
	if s.failNth > 0 && n == s.failNth {
		if s.cancel != nil {
			s.cancel()
		}
		return context.Canceled
	}
	if s.max > 0 && n >= s.max && s.cancel != nil {
		s.cancel()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *recordingSleeper) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// TestRunIntervalLoopRunsManyCycles drives the loop under an injected sleeper so
// no real time passes: it should run a cycle, sleep, repeat, until the context is
// cancelled, then exit 0.
func TestRunIntervalLoopRunsManyCycles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cycles int
	cycle := func(context.Context) error {
		cycles++
		return nil
	}
	sl := &recordingSleeper{cancel: cancel, max: 3}

	var errb bytes.Buffer
	code := runIntervalLoop(ctx, time.Minute, cycle, sl.sleep, &errb)
	if code != 0 {
		t.Fatalf("loop exit = %d, want 0 (stderr: %s)", code, errb.String())
	}
	// With max=3 the loop runs: cycle, sleep(1)->cycle, sleep(2)->cycle, sleep(3)->cancelled.
	if cycles != 3 {
		t.Errorf("ran %d cycles, want 3", cycles)
	}
	if sl.count() != 3 {
		t.Errorf("slept %d times, want 3", sl.count())
	}
	for _, d := range sl.calls {
		if d != time.Minute {
			t.Errorf("slept for %v, want the configured 1m interval", d)
		}
	}
}

// TestRunIntervalLoopCancelDuringSleepExitsClean verifies that a cancel arriving
// during the sleep (between cycles) exits 0 and does NOT start another cycle.
func TestRunIntervalLoopCancelDuringSleepExitsClean(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cycles int
	cycle := func(context.Context) error {
		cycles++
		return nil
	}
	// failNth=1: the very first sleep cancels and returns, so exactly one cycle ran.
	sl := &recordingSleeper{cancel: cancel, failNth: 1}

	var errb bytes.Buffer
	code := runIntervalLoop(ctx, time.Minute, cycle, sl.sleep, &errb)
	if code != 0 {
		t.Fatalf("loop exit = %d, want 0 after cancel during sleep (stderr: %s)", code, errb.String())
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles, want exactly 1 (no new cycle after cancel during sleep)", cycles)
	}
}

// TestRunIntervalLoopCancelMidCycleFinishesCycle verifies that a cancel arriving
// while a cycle is in flight lets that cycle complete, then exits 0 without
// starting another cycle or sleeping again.
func TestRunIntervalLoopCancelMidCycleFinishesCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cycles int
	var finished bool
	cycle := func(context.Context) error {
		cycles++
		// Simulate a signal arriving mid-issuance: cancel, but we keep working.
		cancel()
		finished = true
		return nil
	}
	sl := &recordingSleeper{cancel: cancel}

	var errb bytes.Buffer
	code := runIntervalLoop(ctx, time.Minute, cycle, sl.sleep, &errb)
	if code != 0 {
		t.Fatalf("loop exit = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if !finished {
		t.Error("the in-flight cycle was interrupted; it must finish")
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles, want exactly 1 (cancel mid-cycle then exit)", cycles)
	}
	if sl.count() != 0 {
		t.Errorf("slept %d times after a mid-cycle cancel, want 0 (exit before sleeping)", sl.count())
	}
}

// TestRunIntervalLoopFailedCycleContinues verifies a per-cycle error is logged
// and the loop continues to the next sleep rather than exiting non-zero.
func TestRunIntervalLoopFailedCycleContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cycles int
	cycle := func(context.Context) error {
		cycles++
		return errors.New("transient ACME blip")
	}
	sl := &recordingSleeper{cancel: cancel, max: 2}

	var errb bytes.Buffer
	code := runIntervalLoop(ctx, time.Minute, cycle, sl.sleep, &errb)
	if code != 0 {
		t.Fatalf("loop exit = %d, want 0 — a failed cycle must not exit the loop (stderr: %s)", code, errb.String())
	}
	if cycles != 2 {
		t.Errorf("ran %d cycles, want 2 — the loop must continue after a failed cycle", cycles)
	}
}
