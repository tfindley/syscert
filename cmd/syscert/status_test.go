package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tfindley/syscert/internal/config"
)

func TestWriteStatusShowsCertAndNeverLeaksHMAC(t *testing.T) {
	dir := freshCertStore(t) // 90-day cert, ~89 days left, not due
	if err := os.MkdirAll(filepath.Join(dir, "accounts", "abc"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Cert:    config.CertConfig{Hostname: "host.example.com", KeyType: "ec256"},
		ACME:    config.ACMEConfig{CA: "letsencrypt", Email: "a@example.com", Challenge: "dns-01", EAB: config.ACMEEABConfig{Kid: "kid-123"}},
		Store:   config.StoreConfig{Path: dir},
		Logging: config.LoggingConfig{Level: "info"},
	}
	t.Setenv("SYSCERT_EAB_HMAC", "supersecret-hmac-value")

	var buf bytes.Buffer
	writeStatus(&buf, cfg, time.Now())
	out := buf.String()

	for _, want := range []string{
		"host.example.com",                       // subject + cert CN/SAN
		"EAB:       yes",                         // kid set
		"issued:", "expires:", "renews:     in ", // the dates/durations the user asked for
		"ECDSA",        // key type
		"accounts:  1", // one account dir present
	} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "supersecret-hmac-value") {
		t.Fatal("status output must NEVER contain the EAB HMAC")
	}
}

func TestWriteStatusNoCert(t *testing.T) {
	cfg := &config.Config{
		Cert:    config.CertConfig{Hostname: "host.example.com", KeyType: "ec256"},
		ACME:    config.ACMEConfig{CA: "letsencrypt", Email: "a@example.com", Challenge: "dns-01"},
		Store:   config.StoreConfig{Path: t.TempDir()},
		Logging: config.LoggingConfig{Level: "info"},
	}
	var buf bytes.Buffer
	writeStatus(&buf, cfg, time.Now())
	if !strings.Contains(buf.String(), "none yet") {
		t.Errorf("empty store should report no certificate:\n%s", buf.String())
	}
}

func TestCertLineSummary(t *testing.T) {
	cfg := &config.Config{Store: config.StoreConfig{Path: freshCertStore(t)}}
	s := certLine(cfg, time.Now())
	if !strings.Contains(s, "cert expires") || !strings.Contains(s, "renews") {
		t.Errorf("certLine = %q, want expiry + renewal summary", s)
	}
	// empty store → empty summary
	if certLine(&config.Config{Store: config.StoreConfig{Path: t.TempDir()}}, time.Now()) != "" {
		t.Error("certLine on an empty store should be empty")
	}
}
