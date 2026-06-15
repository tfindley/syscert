package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --env-file populates the process environment before the ACME flow runs, so a
// manual run picks up DNS/CA creds without exporting them by hand.
func TestDryRunLoadsEnvFile(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("SYSCERT_WIRETEST_TOKEN") })
	envp := filepath.Join(t.TempDir(), "secrets")
	if err := os.WriteFile(envp, []byte("SYSCERT_WIRETEST_TOKEN=abc123\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	code := run([]string{"dry-run", "--config-only", "--config", writeCfg(t, validTOML), "--env-file", envp}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if got := os.Getenv("SYSCERT_WIRETEST_TOKEN"); got != "abc123" {
		t.Errorf("--env-file did not set the var: got %q", got)
	}
}

// A bad --env-file fails fast, citing the path, before the config is loaded.
func TestDryRunEnvFileMissingErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	var out, errb bytes.Buffer
	code := run([]string{"dry-run", "--config-only", "--config", writeCfg(t, validTOML), "--env-file", missing}, &out, &errb)
	if code == 0 {
		t.Fatalf("expected non-zero exit for a missing env-file")
	}
	if !strings.Contains(errb.String(), "nope") {
		t.Errorf("error should cite the offending path; got stderr %q", errb.String())
	}
}
