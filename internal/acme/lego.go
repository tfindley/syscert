package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-acme/lego/v5/acme"
	"github.com/go-acme/lego/v5/certcrypto"
	"github.com/go-acme/lego/v5/certificate"
	"github.com/go-acme/lego/v5/challenge/dns01"
	"github.com/go-acme/lego/v5/challenge/http01"
	"github.com/go-acme/lego/v5/challenge/tlsalpn01"
	"github.com/go-acme/lego/v5/lego"
	"github.com/go-acme/lego/v5/providers/dns"
	"github.com/go-acme/lego/v5/registration"
)

// legoObtainer is the production Obtainer backed by lego. It is intentionally
// not unit-tested (it talks to a real ACME server); it is exercised by running
// against a real directory (LE staging / Vault).
type legoObtainer struct{}

// NewLegoObtainer returns the real lego-backed Obtainer.
func NewLegoObtainer() Obtainer { return legoObtainer{} }

// acmeUser implements registration.User.
type acmeUser struct {
	email string
	reg   *acme.ExtendedAccount
	key   crypto.Signer
}

func (u *acmeUser) GetEmail() string                       { return u.email }
func (u *acmeUser) GetRegistration() *acme.ExtendedAccount { return u.reg }
func (u *acmeUser) GetPrivateKey() crypto.Signer           { return u.key }

func (legoObtainer) Obtain(ctx context.Context, p Params) (*Result, error) {
	if p.DirectoryURL == "" {
		return nil, fmt.Errorf("no ACME directory URL resolved (set acme.directory_url)")
	}
	kt, err := certcrypto.ToKeyType(p.KeyType)
	if err != nil {
		return nil, err
	}
	client, err := newClientAndAccount(ctx, p)
	if err != nil {
		return nil, err
	}
	if err := setSolver(client, p); err != nil {
		return nil, err
	}
	res, err := client.Certificate.Obtain(ctx, certificate.ObtainRequest{
		Domains: p.Identifiers,
		KeyType: kt,
		Profile: p.Profile,
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("obtain certificate: %w", err)
	}
	return &Result{
		Identifiers:       res.Domains,
		Certificate:       res.Certificate,
		PrivateKey:        res.PrivateKey,
		IssuerCertificate: res.IssuerCertificate,
	}, nil
}

func (legoObtainer) Revoke(ctx context.Context, p Params, certPEM []byte) error {
	if p.DirectoryURL == "" {
		return fmt.Errorf("no ACME directory URL resolved (set acme.directory_url)")
	}
	client, err := newClientAndAccount(ctx, p)
	if err != nil {
		return err
	}
	if err := client.Certificate.Revoke(ctx, certPEM); err != nil {
		return fmt.Errorf("revoke certificate: %w", err)
	}
	return nil
}

// newClientAndAccount builds a lego client using the account key (persistent
// when p.AccountDir is set, ephemeral otherwise), optionally trusting an
// internal CA for the connection (ca_bundle), and registers/loads the account.
func newClientAndAccount(ctx context.Context, p Params) (*lego.Client, error) {
	key, existed, err := accountSigner(p)
	if err != nil {
		return nil, err
	}
	user := &acmeUser{email: p.Email, key: key}

	cfg := lego.NewConfig(user)
	cfg.CADirURL = p.DirectoryURL

	// Trust the internal CA for this connection only (ADR-0035) — not the system store.
	if p.CABundle != "" {
		pool, err := loadCABundle(p.CABundle)
		if err != nil {
			return nil, err
		}
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
			},
		}
	}

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("acme client: %w", err)
	}

	// A persisted account is reused by *resolving* it (newAccount with
	// onlyReturnExisting) — this re-establishes the account URL for orders without
	// re-sending EAB, so a single-use EAB token is validated only on first
	// registration and never replayed (ADR-0042). A key written before its
	// registration completed falls back to a fresh register.
	var reg *acme.ExtendedAccount
	if existed {
		reg, err = client.Registration.ResolveAccountByKey(ctx)
		if isAccountDoesNotExist(err) {
			reg, err = registerAccount(ctx, client, p)
		}
	} else {
		reg, err = registerAccount(ctx, client, p)
	}
	if err != nil {
		return nil, fmt.Errorf("register account: %w", err)
	}
	user.reg = reg
	return client, nil
}

// registerAccount creates the ACME account, sending External Account Binding when
// configured. Called only on first registration (no existing account to resolve).
func registerAccount(ctx context.Context, client *lego.Client, p Params) (*acme.ExtendedAccount, error) {
	eab, useEAB, err := eabOptions(p)
	if err != nil {
		return nil, err
	}
	if useEAB {
		return client.Registration.RegisterWithExternalAccountBinding(ctx, eab)
	}
	return client.Registration.Register(ctx, registration.RegisterOptions{TermsOfServiceAgreed: true})
}

// isAccountDoesNotExist reports whether err is the ACME "accountDoesNotExist"
// problem — the key isn't registered with this CA yet, so we must register.
func isAccountDoesNotExist(err error) bool {
	var pd *acme.ProblemDetails
	return errors.As(err, &pd) && pd.Type == acme.AccountDoesNotExistErrorType
}

// eabOptions decides whether to use External Account Binding. EAB is on when
// p.EABKid is set; it then requires the HMAC key (p.EABHMAC, from SYSCERT_EAB_HMAC)
// and errors clearly when it's missing — caught before any network call.
func eabOptions(p Params) (opts registration.RegisterEABOptions, useEAB bool, err error) {
	if p.EABKid == "" {
		return registration.RegisterEABOptions{}, false, nil
	}
	if p.EABHMAC == "" {
		return registration.RegisterEABOptions{}, false,
			fmt.Errorf("acme.eab.kid is set but %s is empty (export the EAB HMAC key)", envEABHMAC)
	}
	return registration.RegisterEABOptions{
		TermsOfServiceAgreed: true,
		Kid:                  p.EABKid,
		HmacEncoded:          p.EABHMAC,
	}, true, nil
}

// accountSigner returns the persistent per-CA account key, or an ephemeral one
// when no AccountDir is configured (e.g. dry-run).
func accountSigner(p Params) (crypto.Signer, bool, error) {
	if p.AccountDir == "" {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		return key, false, err // ephemeral: never registered before
	}
	return accountKey(p.AccountDir, p.DirectoryURL)
}

// dnsPropagationOpts maps the configured propagation-check mode to lego's DNS-01
// options. lego's default requires the record to be visible on both the local
// recursive resolver and the CA's authoritative nameservers; "authoritative"
// drops the fragile recursive check (split-horizon / VPN / stale local resolver)
// while still verifying on the authoritative NS, and "off" skips the local check.
func dnsPropagationOpts(mode string) []dns01.ChallengeOption {
	switch mode {
	case "authoritative":
		return []dns01.ChallengeOption{dns01.DisableRecursiveNSsPropagationRequirement()}
	case "off":
		return []dns01.ChallengeOption{
			dns01.DisableRecursiveNSsPropagationRequirement(),
			dns01.DisableAuthoritativeNssPropagationRequirement(),
		}
	default: // "" / "all"
		return nil
	}
}

// setSolver configures the challenge solver for the requested challenge type.
func setSolver(client *lego.Client, p Params) error {
	switch p.Challenge {
	case "dns-01":
		provider, err := dns.NewDNSChallengeProviderByName(p.DNSProvider)
		if err != nil {
			return fmt.Errorf("dns provider %q: %w", p.DNSProvider, err)
		}
		return client.Challenge.SetDNS01Provider(provider, dnsPropagationOpts(p.DNSPropagationCheck)...)
	case "http-01":
		return client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	case "tls-alpn-01":
		return client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", "443"))
	default:
		return fmt.Errorf("unsupported challenge %q", p.Challenge)
	}
}
