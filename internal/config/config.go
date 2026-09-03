// Package config loads and represents the SysCert TOML configuration.
package config

import (
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Config is the top-level SysCert configuration (see docs/config-reference.md).
type Config struct {
	Cert       CertConfig         `toml:"cert"`
	ACME       ACMEConfig         `toml:"acme"`
	Store      StoreConfig        `toml:"store"`
	Bundle     BundleConfig       `toml:"bundle"`
	Distribute []DistributeTarget `toml:"distribute"`
	Renewal    RenewalConfig      `toml:"renewal"`
	Logging    LoggingConfig      `toml:"logging"`
	Observe    ObserveConfig      `toml:"observe"`
}

// CertConfig describes the certificate subject and key.
type CertConfig struct {
	Hostname string   `toml:"hostname"`
	SANs     []string `toml:"sans"`
	IPSANs   []string `toml:"ip_sans"`
	KeyType  string   `toml:"key_type"`
	ReuseKey bool     `toml:"reuse_key"`
}

// ACMEConfig describes the CA and challenge settings.
type ACMEConfig struct {
	CA           string        `toml:"ca"`
	DirectoryURL string        `toml:"directory_url"`
	Email        string        `toml:"email"`
	Challenge    string        `toml:"challenge"`
	Profile      string        `toml:"profile"`
	CABundle     string        `toml:"ca_bundle"`
	DNS          ACMEDNSConfig `toml:"dns"`
	EAB          ACMEEABConfig `toml:"eab"`
}

// ACMEEABConfig holds External Account Binding settings. The Kid (key identifier)
// lives here; the HMAC key is a secret supplied via the SYSCERT_EAB_HMAC env var,
// never in this file. Setting Kid enables EAB at account registration.
type ACMEEABConfig struct {
	Kid string `toml:"kid"`
}

// ACMEDNSConfig holds DNS-challenge settings (credentials come from secrets, not here).
type ACMEDNSConfig struct {
	Provider string `toml:"provider"`
	// PropagationCheck controls lego's DNS-01 propagation pre-check before it
	// notifies the CA: "all" (default — recursive + authoritative nameservers),
	// "authoritative" (skip the local recursive-resolver check; verify only on the
	// CA's authoritative NS — use this when the host's resolver is split-horizon /
	// VPN / slow), or "off" (skip the local check entirely).
	PropagationCheck string `toml:"propagation_check"`
}

// StoreConfig is the canonical store location and permissions.
type StoreConfig struct {
	Path        string `toml:"path"`
	Group       string `toml:"group"`        // optional group granted access to the store dir; "" keeps syscert
	DirMode     string `toml:"dir_mode"`     // octal store-dir mode; default "0700"
	ArchiveKeep int    `toml:"archive_keep"` // previous-cert snapshots to retain; 0 disables history
}

// ParsedDirMode returns the store directory mode, defaulting to 0700 when unset or
// unparseable (validation rejects an unparseable value before issuance).
func (s StoreConfig) ParsedDirMode() os.FileMode {
	if v, err := strconv.ParseUint(s.DirMode, 8, 32); err == nil {
		return os.FileMode(v)
	}
	return 0o700
}

// BundleConfig controls bundle.pem composition.
type BundleConfig struct {
	Order []string `toml:"order"`
}

// DistributeTarget is one cert artifact copied to a consumer path.
type DistributeTarget struct {
	Artifact       string `toml:"artifact"`
	Path           string `toml:"path"`
	Owner          string `toml:"owner"`
	Group          string `toml:"group"`
	Mode           string `toml:"mode"`
	SELinuxContext string `toml:"selinux_context"`
}

// RenewalConfig controls the renewal window.
type RenewalConfig struct {
	RenewBefore string `toml:"renew_before"`
}

// ObserveConfig turns on optional machine-readable state files, written after
// each ensure cycle. Both are OFF by default: an empty path means "don't write
// it". Neither is read back by syscert — they exist purely for monitoring and
// inventory to consume, so a write failure is logged and never fails the run.
type ObserveConfig struct {
	// MetricsFile is a Prometheus node_exporter textfile-collector target, e.g.
	// /var/lib/node_exporter/textfile_collector/syscert.prom (must end .prom).
	MetricsFile string `toml:"metrics_file"`
	// AnsibleFactsFile is an Ansible local-facts target, e.g.
	// /etc/ansible/facts.d/syscert.fact, read back as ansible_local.syscert.
	AnsibleFactsFile string `toml:"ansible_facts_file"`
}

// LoggingConfig controls structured logging.
type LoggingConfig struct {
	Level  string `toml:"level"`  // debug | info (default) | warn | error
	Format string `toml:"format"` // text (default) | json
}

// Load reads and decodes a TOML config file, then applies documented defaults.
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

// IsPublicCA reports whether the CA is publicly trusted (its root ships in
// system trust stores), so it needs no system-trust install. Let's Encrypt is
// the only CA SysCert special-cases (built-in directory URLs + --staging).
func IsPublicCA(ca string) bool { return ca == "letsencrypt" }

// IsInternalCA reports whether the CA is an internal/private ACME server
// (ca = "custom" — e.g. HashiCorp Vault PKI or Smallstep step-ca): its root
// isn't publicly trusted and it requires an explicit directory_url. Defined as
// "set, but not public" so trust/lifecycle paths stay correct even for a CA
// value the validator hasn't vetted.
func IsInternalCA(ca string) bool { return ca != "" && !IsPublicCA(ca) }

// EffectiveChallenge returns the challenge SysCert will actually use. When IP
// SANs are configured, a DNS-based challenge (which RFC 8738 forbids for IP
// identifiers) is auto-switched to http-01; an already IP-compatible challenge
// (http-01 / tls-alpn-01) is kept as configured. (ADR-0015.)
func (c *Config) EffectiveChallenge() string {
	ch := c.ACME.Challenge
	if len(c.Cert.IPSANs) > 0 && ch == "dns-01" {
		return "http-01"
	}
	return ch
}

// applyDefaults fills unset fields with the defaults from docs/decisions.md.
func (c *Config) applyDefaults() {
	if c.Cert.KeyType == "" {
		c.Cert.KeyType = "ec256" // ADR-0018
	}
	if c.ACME.Challenge == "" {
		c.ACME.Challenge = "dns-01" // ADR-0003
	}
	if c.Store.Path == "" {
		c.Store.Path = "/var/lib/syscert" // ADR-0008
	}
	if c.Store.DirMode == "" {
		c.Store.DirMode = "0700" // ADR-0041
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if len(c.Bundle.Order) == 0 {
		c.Bundle.Order = []string{"cert", "chain", "root", "key"} // ADR-0030
	}
}
