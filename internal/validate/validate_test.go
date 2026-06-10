package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tfindley/syscert/internal/config"
)

// baseConfig returns a minimal, valid configuration that individual tests mutate.
func baseConfig() *config.Config {
	return &config.Config{
		Cert:    config.CertConfig{Hostname: "host.example.com", KeyType: "ec256"},
		ACME:    config.ACMEConfig{CA: "letsencrypt", Email: "admin@example.com", Challenge: "dns-01"},
		Store:   config.StoreConfig{Path: "/var/lib/syscert"},
		Bundle:  config.BundleConfig{Order: []string{"cert", "chain", "root", "key"}},
		Logging: config.LoggingConfig{Level: "info"},
	}
}

func hasProblem(ps []Problem, substr string) bool {
	for _, p := range ps {
		if strings.Contains(p.Message, substr) || strings.Contains(p.Field, substr) {
			return true
		}
	}
	return false
}

func TestAcceptsMinimalValidConfig(t *testing.T) {
	if ps := Config(baseConfig()); len(ps) != 0 {
		t.Fatalf("valid config produced problems: %+v", ps)
	}
}

func TestRejectsUnknownChallenge(t *testing.T) {
	c := baseConfig()
	c.ACME.Challenge = "magic-01"
	if ps := Config(c); !hasProblem(ps, "challenge") {
		t.Fatalf("unknown challenge: want a 'challenge' problem, got %+v", ps)
	}
}

func TestAllowsIPSANWithDNSChallenge(t *testing.T) {
	// IP SAN + dns-01 is NOT an error: the challenge auto-switches to http-01
	// (config.EffectiveChallenge / ADR-0015), so the validator must not reject it.
	c := baseConfig()
	c.ACME.CA = "custom"
	c.ACME.DirectoryURL = "https://vault.example.com/v1/pki/acme/directory"
	c.Cert.IPSANs = []string{"10.0.0.5"}
	c.ACME.Challenge = "dns-01"
	if ps := Config(c); len(ps) != 0 {
		t.Fatalf("IP SAN + dns-01 should auto-switch, not error; got %+v", ps)
	}
}

func TestAcceptsIPSANWithHTTPChallenge(t *testing.T) {
	c := baseConfig()
	c.ACME.CA = "custom"
	c.ACME.DirectoryURL = "https://vault.example.com/v1/pki/acme/directory"
	c.Cert.IPSANs = []string{"10.0.0.5"}
	c.ACME.Challenge = "http-01"
	if ps := Config(c); len(ps) != 0 {
		t.Fatalf("IP SAN + http-01 on internal CA should be valid, got %+v", ps)
	}
}

func TestRejectsPrivateIPWithPublicCA(t *testing.T) {
	// Public CAs cannot issue for RFC 1918 addresses (ADR-0016 tier-2).
	c := baseConfig()
	c.ACME.CA = "letsencrypt"
	c.Cert.IPSANs = []string{"10.0.0.5"}
	c.ACME.Challenge = "http-01"
	if ps := Config(c); !hasProblem(ps, "private") {
		t.Fatalf("private IP + public CA: want a 'private' problem, got %+v", ps)
	}
}

func TestRejectsBadArtifact(t *testing.T) {
	c := baseConfig()
	c.Distribute = []config.DistributeTarget{{Artifact: "wat", Path: "/x", Mode: "0644"}}
	if ps := Config(c); !hasProblem(ps, "artifact") {
		t.Fatalf("bad artifact: want an 'artifact' problem, got %+v", ps)
	}
}

func TestRejectsWorldReadableKeyArtifact(t *testing.T) {
	// privkey/bundle hold the key and must never be world-readable (ADR-0030).
	c := baseConfig()
	c.Distribute = []config.DistributeTarget{{Artifact: "privkey", Path: "/x", Mode: "0644"}}
	if ps := Config(c); !hasProblem(ps, "mode") {
		t.Fatalf("world-readable privkey: want a 'mode' problem, got %+v", ps)
	}
}

func TestAcceptsLockedDownKeyArtifact(t *testing.T) {
	c := baseConfig()
	c.Distribute = []config.DistributeTarget{{Artifact: "privkey", Path: "/x", Mode: "0600"}}
	if ps := Config(c); len(ps) != 0 {
		t.Fatalf("0600 privkey should be valid, got %+v", ps)
	}
}

func TestRejectsUnknownBundleToken(t *testing.T) {
	c := baseConfig()
	c.Bundle.Order = []string{"cert", "leaf", "key"}
	if ps := Config(c); !hasProblem(ps, "bundle") {
		t.Fatalf("unknown bundle token: want a 'bundle' problem, got %+v", ps)
	}
}

func TestRejectsBadKeyType(t *testing.T) {
	c := baseConfig()
	c.Cert.KeyType = "ed25519"
	if ps := Config(c); !hasProblem(ps, "key_type") {
		t.Fatalf("bad key_type: want a 'key_type' problem, got %+v", ps)
	}
}

func TestRejectsMissingCABundleFile(t *testing.T) {
	c := baseConfig()
	c.ACME.CABundle = "/no/such/ca.pem"
	if ps := Config(c); !hasProblem(ps, "ca_bundle") {
		t.Fatalf("missing ca_bundle file: want a 'ca_bundle' problem, got %+v", ps)
	}
}

func TestAcceptsReadableCABundleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(p, []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := baseConfig()
	c.ACME.CABundle = p
	if ps := Config(c); hasProblem(ps, "ca_bundle") {
		t.Fatalf("readable ca_bundle: want no 'ca_bundle' problem, got %+v", ps)
	}
}

func TestRejectsUnknownLogFormat(t *testing.T) {
	c := baseConfig()
	c.Logging.Format = "yaml"
	if ps := Config(c); !hasProblem(ps, "logging.format") {
		t.Fatalf("bad log format: want a 'logging.format' problem, got %+v", ps)
	}
}

func TestAcceptsKnownLogFormats(t *testing.T) {
	for _, f := range []string{"", "text", "json"} {
		c := baseConfig()
		c.Logging.Format = f
		if ps := Config(c); hasProblem(ps, "logging.format") {
			t.Errorf("format %q should be valid, got %+v", f, ps)
		}
	}
}

func TestRequiresDirectoryURLForInternalCA(t *testing.T) {
	c := baseConfig()
	c.ACME.CA = "custom"
	c.ACME.DirectoryURL = ""
	if ps := Config(c); !hasProblem(ps, "directory_url") {
		t.Fatalf("custom without directory_url: want a 'directory_url' problem, got %+v", ps)
	}
}

func TestRejectsUnknownCA(t *testing.T) {
	// vault/stepca were collapsed into "custom"; a stale value must be rejected.
	c := baseConfig()
	c.ACME.CA = "vault"
	if ps := Config(c); !hasProblem(ps, "unknown ca") {
		t.Fatalf("ca=%q should be rejected (letsencrypt|custom), got %+v", c.ACME.CA, ps)
	}
}

func TestPropagationCheck(t *testing.T) {
	for _, v := range []string{"", "all", "authoritative", "off"} {
		c := baseConfig()
		c.ACME.DNS.PropagationCheck = v
		if ps := Config(c); hasProblem(ps, "propagation_check") {
			t.Errorf("propagation_check=%q should be valid, got %+v", v, ps)
		}
	}
	c := baseConfig()
	c.ACME.DNS.PropagationCheck = "sometimes"
	if ps := Config(c); !hasProblem(ps, "propagation_check") {
		t.Errorf("propagation_check=%q should be rejected, got %+v", c.ACME.DNS.PropagationCheck, ps)
	}
}
