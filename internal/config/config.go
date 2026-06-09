// Package config loads and represents the SysCert TOML configuration.
package config

import "github.com/BurntSushi/toml"

// Config is the top-level SysCert configuration (see docs/config-reference.md).
type Config struct {
	Cert       CertConfig         `toml:"cert"`
	ACME       ACMEConfig         `toml:"acme"`
	Store      StoreConfig        `toml:"store"`
	Bundle     BundleConfig       `toml:"bundle"`
	Distribute []DistributeTarget `toml:"distribute"`
	Renewal    RenewalConfig      `toml:"renewal"`
	Logging    LoggingConfig      `toml:"logging"`
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
	DNS          ACMEDNSConfig `toml:"dns"`
}

// ACMEDNSConfig holds DNS-challenge settings (credentials come from secrets, not here).
type ACMEDNSConfig struct {
	Provider string `toml:"provider"`
}

// StoreConfig is the canonical store location.
type StoreConfig struct {
	Path string `toml:"path"`
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

// LoggingConfig controls logging.
type LoggingConfig struct {
	Level string `toml:"level"`
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

// EffectiveChallenge returns the challenge SysCert will actually use. When IP
// SANs are configured, a DNS-based challenge (which RFC 8738 forbids for IP
// identifiers) is auto-switched to http-01; an already IP-compatible challenge
// (http-01 / tls-alpn-01) is kept as configured. (ADR-0015.)
func (c *Config) EffectiveChallenge() string {
	ch := c.ACME.Challenge
	if len(c.Cert.IPSANs) > 0 && (ch == "dns-01" || ch == "dns-persist-01") {
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
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if len(c.Bundle.Order) == 0 {
		c.Bundle.Order = []string{"cert", "chain", "root", "key"} // ADR-0030
	}
}
