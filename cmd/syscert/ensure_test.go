package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestEnsureOneShotDefault confirms the default (no --interval) path is unchanged:
// a single ensure cycle runs and the process exits. A fresh, not-due cert keeps
// this offline (no ACME), and with no targets distribute is a no-op.
func TestEnsureOneShotDefault(t *testing.T) {
	store := freshCertStore(t)
	cfg := writeCfg(t, validTOML+"\n[store]\npath = \""+store+"\"\n")
	var out, errb bytes.Buffer
	// Bare `syscert` (no --interval) → exactly one ensure cycle, then exit.
	code := run([]string{"--config", cfg}, &out, &errb)
	if code != 0 {
		t.Fatalf("ensure one-shot: exit = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "not due for renewal") {
		t.Errorf("ensure one-shot: want 'not due for renewal', got %q", out.String())
	}
	if !strings.Contains(out.String(), "ensured") {
		t.Errorf("ensure one-shot: want the OK/ensured summary, got %q", out.String())
	}
}

// TestEnsureIntervalStartupValidationErrorExits verifies that a fatal config
// problem with --interval set exits non-zero BEFORE the loop starts (validation
// runs once up front). A sub-1m interval is also a fatal startup error.
func TestEnsureIntervalStartupValidationErrorExits(t *testing.T) {
	t.Run("invalid config exits non-zero before looping", func(t *testing.T) {
		cfg := writeCfg(t, invalidTOML)
		var out, errb bytes.Buffer
		code := run([]string{"--interval", "1m", "--config", cfg}, &out, &errb)
		if code == 0 {
			t.Fatalf("invalid config with --interval: want non-zero exit, got 0")
		}
		combined := out.String() + errb.String()
		if !strings.Contains(combined, "challenge") {
			t.Errorf("want the challenge validation problem; got %q", combined)
		}
	})

	t.Run("sub-1m interval rejected up front", func(t *testing.T) {
		store := freshCertStore(t)
		cfg := writeCfg(t, validTOML+"\n[store]\npath = \""+store+"\"\n")
		var out, errb bytes.Buffer
		code := run([]string{"--interval", "30s", "--config", cfg}, &out, &errb)
		if code == 0 {
			t.Fatalf("--interval 30s: want non-zero exit, got 0")
		}
		if !strings.Contains(errb.String(), "1m") {
			t.Errorf("want an error mentioning the 1m minimum; got stderr %q", errb.String())
		}
	})
}
