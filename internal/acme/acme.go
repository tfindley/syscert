// Package acme drives ACME issuance via lego. The dry-run path runs the full
// order + challenge flow but persists nothing (like `certbot --dry-run`).
package acme

import (
	"context"
	"path/filepath"

	"github.com/tfindley/syscert/internal/config"
)

const (
	leProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	leStagingURL    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// Params is the fully-resolved input to an issuance attempt.
type Params struct {
	DirectoryURL string
	Email        string
	KeyType      string
	Challenge    string
	DNSProvider  string
	Profile      string
	CABundle     string
	AccountDir   string // base dir for the persistent ACME account; "" = ephemeral
	Identifiers  []string
}

// Result reports the outcome of an issuance attempt. The PEM fields carry the
// obtained material (used by `issue`; ignored/discarded by dry-run).
type Result struct {
	Identifiers       []string
	Certificate       []byte // PEM: leaf + intermediates
	PrivateKey        []byte // PEM private key
	IssuerCertificate []byte // PEM intermediate(s)
}

// Obtainer performs the ACME order + challenge, and revocation. Implemented by
// the real lego client; faked in tests.
type Obtainer interface {
	Obtain(ctx context.Context, p Params) (*Result, error)
	Revoke(ctx context.Context, p Params, certPEM []byte) error
}

// DirectoryURL resolves the ACME directory endpoint to use.
//
// An explicit acme.directory_url always wins. Otherwise, for Let's Encrypt it
// selects staging when staging is true and production otherwise. Internal CAs
// must supply directory_url (the validator enforces this).
func DirectoryURL(cfg *config.Config, staging bool) string {
	if cfg.ACME.DirectoryURL != "" {
		return cfg.ACME.DirectoryURL
	}
	if cfg.ACME.CA == "letsencrypt" {
		if staging {
			return leStagingURL
		}
		return leProductionURL
	}
	return ""
}

// Identifiers returns the certificate identifiers: the subject first, then any
// DNS SANs and IP SANs, de-duplicated. lego treats IP-shaped entries as IP
// identifiers (RFC 8738).
func Identifiers(cfg *config.Config, subject string) []string {
	out := make([]string, 0, 1+len(cfg.Cert.SANs)+len(cfg.Cert.IPSANs))
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	add(subject)
	for _, id := range cfg.Cert.SANs {
		add(id)
	}
	for _, id := range cfg.Cert.IPSANs {
		add(id)
	}
	return out
}

// params builds the resolved Params. accountDir is the persistent-account base
// ("" for an ephemeral account, e.g. dry-run).
func params(cfg *config.Config, subject string, staging bool, accountDir string) Params {
	return Params{
		DirectoryURL: DirectoryURL(cfg, staging),
		Email:        cfg.ACME.Email,
		KeyType:      cfg.Cert.KeyType,
		Challenge:    cfg.EffectiveChallenge(),
		DNSProvider:  cfg.ACME.DNS.Provider,
		Profile:      cfg.ACME.Profile,
		CABundle:     cfg.ACME.CABundle,
		AccountDir:   accountDir,
		Identifiers:  Identifiers(cfg, subject),
	}
}

func accountsDir(cfg *config.Config) string {
	return filepath.Join(cfg.Store.Path, "accounts")
}

// Obtain runs the full ACME order + challenge through the obtainer and returns
// the result (including cert material). It uses the persistent per-CA account.
// staging selects the CA's staging directory (Let's Encrypt only).
func Obtain(ctx context.Context, cfg *config.Config, subject string, ob Obtainer, staging bool) (*Result, error) {
	return ob.Obtain(ctx, params(cfg, subject, staging, accountsDir(cfg)))
}

// DryRun runs the full ACME flow against the staging directory with an ephemeral
// account and discards the result's cert material — the `certbot --dry-run`
// equivalent (nothing is persisted).
func DryRun(ctx context.Context, cfg *config.Config, subject string, ob Obtainer) (*Result, error) {
	return ob.Obtain(ctx, params(cfg, subject, true, ""))
}

// RevokeCert revokes a previously-issued certificate via the persistent account.
func RevokeCert(ctx context.Context, cfg *config.Config, certPEM []byte, ob Obtainer, staging bool) error {
	return ob.Revoke(ctx, params(cfg, "", staging, accountsDir(cfg)), certPEM)
}
