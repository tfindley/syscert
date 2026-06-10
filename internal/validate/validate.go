// Package validate runs SysCert's fail-fast configuration checks (ADR-0016/0026).
//
// These are the deterministic, locally-knowable rules — spec invariants (tier 1)
// and facts derivable from config alone (tier 2). CA-capability discovery
// (tier 3) happens at runtime, not here.
package validate

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/tfindley/syscert/internal/config"
)

// Problem is a single validation failure with the field it concerns.
type Problem struct {
	Field   string
	Message string
}

var (
	validChallenges = map[string]bool{"dns-01": true, "http-01": true, "tls-alpn-01": true, "dns-persist-01": true}
	validKeyTypes   = map[string]bool{"ec256": true, "ec384": true, "rsa2048": true, "rsa4096": true}
	validArtifacts  = map[string]bool{"cert": true, "privkey": true, "chain": true, "fullchain": true, "bundle": true}
	keyBearing      = map[string]bool{"privkey": true, "bundle": true}
	validBundleTok  = map[string]bool{"cert": true, "chain": true, "root": true, "key": true}
	validLogFormats = map[string]bool{"": true, "text": true, "json": true} // "" => default (text)
)

// Config checks the configuration and returns any problems (empty == valid).
func Config(cfg *config.Config) []Problem {
	var ps []Problem
	add := func(field, msg string) { ps = append(ps, Problem{Field: field, Message: msg}) }

	// Required CA fields.
	if cfg.ACME.CA == "" {
		add("acme.ca", "a CA must be set (e.g. letsencrypt, vault, stepca)")
	}
	if cfg.ACME.Email == "" {
		add("acme.email", "an ACME account email is required")
	}
	if config.IsInternalCA(cfg.ACME.CA) && cfg.ACME.DirectoryURL == "" {
		add("acme.directory_url", fmt.Sprintf("ca=%q requires acme.directory_url", cfg.ACME.CA))
	}
	if cfg.ACME.CABundle != "" {
		if _, err := os.Stat(cfg.ACME.CABundle); err != nil {
			add("acme.ca_bundle", fmt.Sprintf("cannot read ca_bundle file %q: %v", cfg.ACME.CABundle, err))
		}
	}

	if !validLogFormats[cfg.Logging.Format] {
		add("logging.format", fmt.Sprintf("unknown format %q (text|json)", cfg.Logging.Format))
	}

	// Challenge + key type are from fixed sets.
	if !validChallenges[cfg.ACME.Challenge] {
		add("acme.challenge", fmt.Sprintf("unknown challenge %q", cfg.ACME.Challenge))
	}
	if !validKeyTypes[cfg.Cert.KeyType] {
		add("cert.key_type", fmt.Sprintf("unknown key_type %q (ec256|ec384|rsa2048|rsa4096)", cfg.Cert.KeyType))
	}

	// IP SANs: a DNS challenge auto-switches to http-01 (config.EffectiveChallenge),
	// so the only IP-SAN rule left here is the public-CA / routability policy.
	if len(cfg.Cert.IPSANs) > 0 {
		public := config.IsPublicCA(cfg.ACME.CA)
		for _, raw := range cfg.Cert.IPSANs {
			ip := net.ParseIP(raw)
			if ip == nil {
				add("cert.ip_sans", fmt.Sprintf("%q is not a valid IP address", raw))
				continue
			}
			if public && !isPubliclyRoutable(ip) {
				add("cert.ip_sans", fmt.Sprintf("%q is a private/non-routable IP; public CAs cannot issue for it — use an internal CA", raw))
			}
		}
	}

	// bundle.order tokens.
	seen := map[string]bool{}
	for _, tok := range cfg.Bundle.Order {
		if !validBundleTok[tok] {
			add("bundle.order", fmt.Sprintf("unknown component %q (cert|chain|root|key)", tok))
		}
		if seen[tok] {
			add("bundle.order", fmt.Sprintf("duplicate component %q", tok))
		}
		seen[tok] = true
	}

	// Distribution targets.
	for i, d := range cfg.Distribute {
		where := fmt.Sprintf("distribute[%d]", i)
		if !validArtifacts[d.Artifact] {
			add(where+".artifact", fmt.Sprintf("unknown artifact %q (cert|privkey|chain|fullchain|bundle)", d.Artifact))
		}
		if keyBearing[d.Artifact] && d.Mode != "" && worldReadable(d.Mode) {
			add(where+".mode", fmt.Sprintf("artifact %q holds the private key; mode %q is world-readable", d.Artifact, d.Mode))
		}
	}

	return ps
}

// isPubliclyRoutable reports whether an address could plausibly be issued by a public CA.
func isPubliclyRoutable(ip net.IP) bool {
	return !(ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified())
}

// worldReadable reports whether an octal mode string grants other-read.
func worldReadable(mode string) bool {
	v, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return false // unparseable modes are caught elsewhere; don't double-report here
	}
	return v&0o004 != 0
}
