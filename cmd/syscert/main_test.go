package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validTOML = `
[cert]
hostname = "host.example.com"

[acme]
ca        = "letsencrypt"
email     = "admin@example.com"
challenge = "dns-01"
`

const invalidTOML = `
[cert]
hostname = "host.example.com"

[acme]
ca        = "letsencrypt"
email     = "admin@example.com"
challenge = "magic-01"
`

func writeCfg(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "syscert.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDryRunConfigOnlyValidConfig(t *testing.T) {
	// --config-only keeps this offline: config test + subject resolution, no ACME.
	var out, errb bytes.Buffer
	code := run([]string{"dry-run", "--config-only", "--config", writeCfg(t, validTOML)}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "host.example.com") {
		t.Errorf("output should report the resolved subject; got %q", out.String())
	}
}

func TestDryRunInvalidConfig(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"dry-run", "--config", writeCfg(t, invalidTOML)}, &out, &errb)
	if code == 0 {
		t.Fatalf("invalid config: want non-zero exit, got 0")
	}
	combined := out.String() + errb.String()
	if !strings.Contains(combined, "challenge") {
		t.Errorf("output should mention the challenge problem; got %q", combined)
	}
}

func TestStubCommandNotImplemented(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"issue", "--config", writeCfg(t, validTOML)}, &out, &errb)
	if code == 0 {
		t.Fatalf("stub command: want non-zero exit")
	}
	if !strings.Contains(out.String()+errb.String(), "not implemented") {
		t.Errorf("stub should say 'not implemented'; got %q", out.String()+errb.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"bogus"}, &out, &errb); code == 0 {
		t.Fatal("unknown command: want non-zero exit")
	}
}

func TestVersionCommand(t *testing.T) {
	for _, arg := range []string{"version", "--version"} {
		var out, errb bytes.Buffer
		code := run([]string{arg}, &out, &errb)
		if code != 0 {
			t.Fatalf("%q: exit = %d, want 0", arg, code)
		}
		if !strings.Contains(out.String(), version) {
			t.Errorf("%q: output %q should contain version %q", arg, out.String(), version)
		}
	}
}
