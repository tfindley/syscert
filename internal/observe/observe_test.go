package observe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sample() Snapshot {
	return Snapshot{
		Version:    "v0.4.0",
		Subject:    "host.example.com",
		CA:         "letsencrypt",
		Challenge:  "dns-01",
		HasCert:    true,
		KeyType:    "ECDSA P-256",
		Issuer:     "CN=Example R3",
		Serial:     "04:9a:bc",
		NotBefore:  time.Unix(1_700_000_000, 0),
		NotAfter:   time.Unix(1_707_776_000, 0),
		RenewAfter: time.Unix(1_705_184_000, 0),
		RenewalDue: false,
		Targets: []Target{
			{Artifact: "fullchain", Path: "/etc/nginx/tls/fullchain.pem", Present: true},
			{Artifact: "cert", Path: "/etc/cockpit/ws-certs.d/99-syscert.crt", Present: false},
		},
		Generated: time.Unix(1_705_000_000, 0),
	}
}

func TestPrometheusEmitsExpectedSeries(t *testing.T) {
	out := string(Prometheus(sample()))
	for _, want := range []string{
		"syscert_cert_not_after_seconds 1707776000",
		"syscert_cert_present 1",
		"syscert_cert_renewal_due 0",
		"syscert_distribute_targets 2",
		"syscert_distribute_targets_present 1",
		`syscert_distribute_target_present{path="/etc/nginx/tls/fullchain.pem",artifact="fullchain"} 1`,
		`syscert_distribute_target_present{path="/etc/cockpit/ws-certs.d/99-syscert.crt",artifact="cert"} 0`,
		`subject="host.example.com"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	// Every metric needs its HELP/TYPE pair or promtool rejects the file.
	for _, name := range []string{"syscert_cert_not_after_seconds", "syscert_cert_info", "syscert_distribute_target_present"} {
		if !strings.Contains(out, "# HELP "+name+" ") || !strings.Contains(out, "# TYPE "+name+" gauge") {
			t.Errorf("metric %s is missing its HELP/TYPE lines", name)
		}
	}
}

// A quote or backslash in a label would otherwise break every following line of
// the file, so node_exporter would drop the lot.
func TestPrometheusEscapesLabelValues(t *testing.T) {
	s := sample()
	s.Issuer = `CN=We "Quote" \ Things`
	s.Targets = []Target{{Artifact: "cert", Path: `/etc/od"d\path`, Present: true}}
	out := string(Prometheus(s))

	if !strings.Contains(out, `issuer="CN=We \"Quote\" \\ Things"`) {
		t.Errorf("issuer label not escaped:\n%s", out)
	}
	if !strings.Contains(out, `path="/etc/od\"d\\path"`) {
		t.Errorf("path label not escaped:\n%s", out)
	}
	// Exactly one line per metric family: an unescaped quote would split lines.
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, " ") {
			t.Errorf("malformed sample line %q", line)
		}
	}
}

func TestPrometheusOmitsCertSeriesWhenAbsent(t *testing.T) {
	out := string(Prometheus(Snapshot{Generated: time.Unix(1, 0)}))
	if strings.Contains(out, "syscert_cert_not_after_seconds") {
		t.Error("expiry series emitted with no certificate present")
	}
	for _, want := range []string{"syscert_cert_present 0", "syscert_distribute_targets 0"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestAnsibleFactsIsValidJSONWithTargets(t *testing.T) {
	data, err := AnsibleFacts(sample())
	if err != nil {
		t.Fatalf("AnsibleFacts: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
	if got["subject"] != "host.example.com" || got["ca"] != "letsencrypt" {
		t.Errorf("unexpected facts: %v", got)
	}
	targets, ok := got["targets"].([]any)
	if !ok || len(targets) != 2 {
		t.Fatalf("targets = %v, want 2 entries", got["targets"])
	}
}

// Ansible filters on ansible_local.syscert.targets; null would break them.
func TestAnsibleFactsRendersEmptyTargetsAsArray(t *testing.T) {
	data, err := AnsibleFacts(Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"targets": []`) {
		t.Errorf("empty targets should render as [], got:\n%s", data)
	}
}

func TestWriteIsAtomicAndWorldReadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "syscert.prom")
	if err := Write(path, []byte("x 1\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %#o, want 0644 (node_exporter and Ansible are not the syscert user)", fi.Mode().Perm())
	}
	// Overwriting must leave no temp files behind for the collector to trip over.
	if err := Write(path, []byte("x 2\n")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the target file, found %d entries", len(entries))
	}
}
