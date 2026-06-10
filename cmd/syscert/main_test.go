package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tfindley/syscert/internal/config"
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

// freshCertStore writes a store dir containing a not-due cert.pem (90-day cert,
// most of its life remaining) so renew/ensure can be exercised offline.
func freshCertStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-1 * 24 * time.Hour),
		NotAfter:     time.Now().Add(89 * 24 * time.Hour),
		DNSNames:     []string{"host.example.com"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, "cert.pem"), pemBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRenewNotDueIsNoOp(t *testing.T) {
	// A fresh cert is not due; renew must short-circuit (exit 0) before any ACME call.
	store := freshCertStore(t)
	cfg := writeCfg(t, validTOML+"\n[store]\npath = \""+store+"\"\n")
	var out, errb bytes.Buffer
	code := run([]string{"renew", "--config", cfg}, &out, &errb)
	if code != 0 {
		t.Fatalf("renew not-due: exit = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "not due") {
		t.Errorf("renew not-due: want 'not due' message, got %q", out.String())
	}
}

func TestConfirm(t *testing.T) {
	cases := []struct {
		in    string
		force bool
		want  bool
	}{
		{"y\n", false, true},
		{"yes\n", false, true},
		{"Y\n", false, true},
		{"n\n", false, false},
		{"\n", false, false},
		{"", false, false}, // EOF → No
		{"", true, true},   // force short-circuits without reading
	}
	for _, c := range cases {
		var out bytes.Buffer
		if got := confirm(&out, strings.NewReader(c.in), "proceed? ", c.force); got != c.want {
			t.Errorf("confirm(in=%q, force=%v) = %v, want %v", c.in, c.force, got, c.want)
		}
	}
}

func TestNoteConnectionTrust(t *testing.T) {
	var withBundle bytes.Buffer
	noteConnectionTrust(&config.Config{ACME: config.ACMEConfig{CABundle: "/etc/syscert/ca.pem"}}, &withBundle)
	if !strings.Contains(withBundle.String(), "ca_bundle") || !strings.Contains(withBundle.String(), "warning") {
		t.Errorf("ca_bundle set: want a warning mentioning ca_bundle, got %q", withBundle.String())
	}
	var noBundle bytes.Buffer
	noteConnectionTrust(&config.Config{}, &noBundle)
	if noBundle.Len() != 0 {
		t.Errorf("no ca_bundle: want no output, got %q", noBundle.String())
	}
}

func TestRenewNoCertErrors(t *testing.T) {
	store := t.TempDir() // empty store, no cert.pem
	cfg := writeCfg(t, validTOML+"\n[store]\npath = \""+store+"\"\n")
	var out, errb bytes.Buffer
	if code := run([]string{"renew", "--config", cfg}, &out, &errb); code == 0 {
		t.Fatal("renew with no stored cert: want non-zero exit")
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
