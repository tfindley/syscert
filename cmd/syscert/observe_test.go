package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tfindley/syscert/internal/config"
)

// obsConfig builds a config whose observe outputs land in a temp directory, with
// one distribution target that exists and one that does not.
func obsConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	store := freshCertStore(t)
	out := t.TempDir()
	present := filepath.Join(out, "there.pem")
	if err := os.WriteFile(present, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		Cert:  config.CertConfig{Hostname: "host.example.com", KeyType: "ec256"},
		ACME:  config.ACMEConfig{CA: "letsencrypt", Email: "a@example.com", Challenge: "dns-01"},
		Store: config.StoreConfig{Path: store},
		Distribute: []config.DistributeTarget{
			{Artifact: "fullchain", Path: present},
			{Artifact: "cert", Path: filepath.Join(out, "missing.pem")},
		},
		Observe: config.ObserveConfig{
			MetricsFile:      filepath.Join(out, "syscert.prom"),
			AnsibleFactsFile: filepath.Join(out, "syscert.fact"),
		},
	}, out
}

func TestWriteObservationsProducesBothFiles(t *testing.T) {
	cfg, out := obsConfig(t)
	var errBuf bytes.Buffer
	writeObservations(cfg, "host.example.com", time.Now(), &errBuf)

	if errBuf.Len() != 0 {
		t.Fatalf("unexpected warnings: %s", errBuf.String())
	}

	prom, err := os.ReadFile(filepath.Join(out, "syscert.prom"))
	if err != nil {
		t.Fatalf("metrics file: %v", err)
	}
	for _, want := range []string{
		"syscert_cert_present 1",
		"syscert_distribute_targets 2",
		"syscert_distribute_targets_present 1", // one target file exists, one doesn't
		"syscert_cert_not_after_seconds ",
		`subject="host.example.com"`,
	} {
		if !strings.Contains(string(prom), want) {
			t.Errorf("metrics missing %q:\n%s", want, prom)
		}
	}

	factsRaw, err := os.ReadFile(filepath.Join(out, "syscert.fact"))
	if err != nil {
		t.Fatalf("facts file: %v", err)
	}
	var facts map[string]any
	if err := json.Unmarshal(factsRaw, &facts); err != nil {
		t.Fatalf("facts are not valid JSON (Ansible would ignore the file): %v\n%s", err, factsRaw)
	}
	if facts["has_cert"] != true {
		t.Errorf("has_cert = %v, want true", facts["has_cert"])
	}
	if facts["subject"] != "host.example.com" {
		t.Errorf("subject = %v", facts["subject"])
	}
}

// Off by default: with no [observe] paths set, nothing is written anywhere.
func TestWriteObservationsIsNoOpWhenUnset(t *testing.T) {
	cfg, out := obsConfig(t)
	cfg.Observe = config.ObserveConfig{}
	before, _ := os.ReadDir(out)

	var errBuf bytes.Buffer
	writeObservations(cfg, "host.example.com", time.Now(), &errBuf)

	after, _ := os.ReadDir(out)
	if len(after) != len(before) {
		t.Errorf("wrote %d new file(s) with observability disabled", len(after)-len(before))
	}
	if errBuf.Len() != 0 {
		t.Errorf("unexpected output: %s", errBuf.String())
	}
}

// An unwritable output must warn, not fail the run — a renewal that succeeded
// must not be reported as a failure because monitoring plumbing is broken.
func TestWriteObservationsWarnsButDoesNotFail(t *testing.T) {
	cfg, _ := obsConfig(t)
	cfg.Observe.MetricsFile = "/nonexistent-dir-for-test/syscert.prom"
	cfg.Observe.AnsibleFactsFile = ""

	var errBuf bytes.Buffer
	writeObservations(cfg, "host.example.com", time.Now(), &errBuf)

	if !strings.Contains(errBuf.String(), "warning: write metrics") {
		t.Errorf("expected a warning, got: %q", errBuf.String())
	}
}

func TestGatherObservationWithoutCert(t *testing.T) {
	cfg := &config.Config{
		ACME:  config.ACMEConfig{CA: "letsencrypt", Challenge: "dns-01"},
		Store: config.StoreConfig{Path: t.TempDir()}, // empty store
	}
	s := gatherObservation(cfg, "host.example.com", time.Now())
	if s.HasCert {
		t.Error("HasCert = true for an empty store")
	}
	if s.Subject != "host.example.com" || s.CA != "letsencrypt" {
		t.Errorf("snapshot lost config context: %+v", s)
	}
}
