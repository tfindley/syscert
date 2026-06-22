package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "syscert.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return p
}

func TestLoadAppliesDefaults(t *testing.T) {
	// A minimal config should come back with the documented defaults filled in.
	p := writeTemp(t, `
[acme]
ca = "letsencrypt"
email = "admin@example.com"
`)

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Cert.KeyType != "ec256" {
		t.Errorf("Cert.KeyType default = %q, want %q", cfg.Cert.KeyType, "ec256")
	}
	if cfg.ACME.Challenge != "dns-01" {
		t.Errorf("ACME.Challenge default = %q, want %q", cfg.ACME.Challenge, "dns-01")
	}
	if cfg.Store.Path != "/var/lib/syscert" {
		t.Errorf("Store.Path default = %q, want %q", cfg.Store.Path, "/var/lib/syscert")
	}
	if cfg.Store.DirMode != "0700" {
		t.Errorf("Store.DirMode default = %q, want %q", cfg.Store.DirMode, "0700")
	}
	if cfg.Store.ArchiveKeep != 0 {
		t.Errorf("Store.ArchiveKeep default = %d, want 0", cfg.Store.ArchiveKeep)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level default = %q, want %q", cfg.Logging.Level, "info")
	}
	wantOrder := []string{"cert", "chain", "root", "key"}
	if got := cfg.Bundle.Order; !equalStrings(got, wantOrder) {
		t.Errorf("Bundle.Order default = %v, want %v", got, wantOrder)
	}
}

func TestLoadParsesStoreOptions(t *testing.T) {
	p := writeTemp(t, `
[acme]
ca = "letsencrypt"
email = "admin@example.com"

[store]
path         = "/srv/syscert"
group        = "ssl-cert"
dir_mode     = "0750"
archive_keep = 5
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Store.Group != "ssl-cert" || cfg.Store.DirMode != "0750" || cfg.Store.ArchiveKeep != 5 {
		t.Errorf("store opts = %+v, want group=ssl-cert dir_mode=0750 archive_keep=5", cfg.Store)
	}
	if m := cfg.Store.ParsedDirMode(); m != 0o750 {
		t.Errorf("ParsedDirMode = %#o, want 0750", m)
	}
}

func TestParsedDirModeFallback(t *testing.T) {
	if m := (StoreConfig{}).ParsedDirMode(); m != 0o700 {
		t.Errorf("empty ParsedDirMode = %#o, want 0700", m)
	}
	if m := (StoreConfig{DirMode: "nonsense"}).ParsedDirMode(); m != 0o700 {
		t.Errorf("invalid ParsedDirMode = %#o, want 0700 fallback", m)
	}
}

func TestLoadParsesValues(t *testing.T) {
	p := writeTemp(t, `
[cert]
hostname = "host.example.com"
sans     = ["alt.example.com"]
ip_sans  = ["10.0.0.5"]
key_type = "rsa2048"

[acme]
ca        = "custom"
challenge = "http-01"
email     = "admin@example.com"

[acme.dns]
provider = "cloudflare"
propagation_check = "authoritative"

[store]
path = "/srv/syscert"

[bundle]
order = ["key", "cert", "chain"]

[[distribute]]
artifact = "fullchain"
path     = "/etc/nginx/tls/fullchain.pem"
owner    = "root"
group    = "root"
mode     = "0644"

[[distribute]]
artifact = "privkey"
path     = "/etc/nginx/tls/privkey.pem"
mode     = "0600"
`)

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Cert.Hostname != "host.example.com" {
		t.Errorf("Cert.Hostname = %q", cfg.Cert.Hostname)
	}
	if cfg.Cert.KeyType != "rsa2048" {
		t.Errorf("Cert.KeyType = %q, want rsa2048 (no default override)", cfg.Cert.KeyType)
	}
	if !equalStrings(cfg.Cert.IPSANs, []string{"10.0.0.5"}) {
		t.Errorf("Cert.IPSANs = %v", cfg.Cert.IPSANs)
	}
	if cfg.ACME.DNS.Provider != "cloudflare" {
		t.Errorf("ACME.DNS.Provider = %q", cfg.ACME.DNS.Provider)
	}
	if cfg.ACME.DNS.PropagationCheck != "authoritative" {
		t.Errorf("ACME.DNS.PropagationCheck = %q, want authoritative", cfg.ACME.DNS.PropagationCheck)
	}
	if cfg.Store.Path != "/srv/syscert" {
		t.Errorf("Store.Path = %q (explicit value should win over default)", cfg.Store.Path)
	}
	if !equalStrings(cfg.Bundle.Order, []string{"key", "cert", "chain"}) {
		t.Errorf("Bundle.Order = %v", cfg.Bundle.Order)
	}
	if len(cfg.Distribute) != 2 {
		t.Fatalf("len(Distribute) = %d, want 2", len(cfg.Distribute))
	}
	if cfg.Distribute[0].Artifact != "fullchain" || cfg.Distribute[1].Artifact != "privkey" {
		t.Errorf("Distribute artifacts = %q, %q", cfg.Distribute[0].Artifact, cfg.Distribute[1].Artifact)
	}
}

func TestLoadMissingFileErrors(t *testing.T) {
	if _, err := Load("/nonexistent/syscert.toml"); err == nil {
		t.Fatal("Load of missing file: want error, got nil")
	}
}

func TestEffectiveChallenge(t *testing.T) {
	cases := []struct {
		name      string
		challenge string
		ipSANs    []string
		want      string
	}{
		{"no ip sans keeps dns-01", "dns-01", nil, "dns-01"},
		{"ip san switches dns-01 to http-01", "dns-01", []string{"10.0.0.5"}, "http-01"},
		{"ip san keeps tls-alpn-01", "tls-alpn-01", []string{"10.0.0.5"}, "tls-alpn-01"},
		{"ip san keeps http-01", "http-01", []string{"10.0.0.5"}, "http-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				Cert: CertConfig{IPSANs: tc.ipSANs},
				ACME: ACMEConfig{Challenge: tc.challenge},
			}
			if got := c.EffectiveChallenge(); got != tc.want {
				t.Errorf("EffectiveChallenge() = %q, want %q", got, tc.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
