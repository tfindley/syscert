package validate

import (
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
	c.ACME.CA = "vault"
	c.ACME.DirectoryURL = "https://vault.example.com/v1/pki/acme/directory"
	c.Cert.IPSANs = []string{"10.0.0.5"}
	c.ACME.Challenge = "dns-01"
	if ps := Config(c); len(ps) != 0 {
		t.Fatalf("IP SAN + dns-01 should auto-switch, not error; got %+v", ps)
	}
}

func TestAcceptsIPSANWithHTTPChallenge(t *testing.T) {
	c := baseConfig()
	c.ACME.CA = "vault"
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

func TestRequiresDirectoryURLForInternalCA(t *testing.T) {
	c := baseConfig()
	c.ACME.CA = "vault"
	c.ACME.DirectoryURL = ""
	if ps := Config(c); !hasProblem(ps, "directory_url") {
		t.Fatalf("vault without directory_url: want a 'directory_url' problem, got %+v", ps)
	}
}
