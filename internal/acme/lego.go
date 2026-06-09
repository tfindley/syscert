package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"

	"github.com/go-acme/lego/v5/acme"
	"github.com/go-acme/lego/v5/certcrypto"
	"github.com/go-acme/lego/v5/certificate"
	"github.com/go-acme/lego/v5/challenge/http01"
	"github.com/go-acme/lego/v5/challenge/tlsalpn01"
	"github.com/go-acme/lego/v5/lego"
	"github.com/go-acme/lego/v5/providers/dns"
	"github.com/go-acme/lego/v5/registration"
)

// legoObtainer is the production Obtainer backed by lego. It is intentionally
// not unit-tested (it talks to a real ACME server); it is exercised by running
// `syscert dry-run` against a real directory (LE staging / Vault).
type legoObtainer struct{}

// NewLegoObtainer returns the real lego-backed Obtainer.
func NewLegoObtainer() Obtainer { return legoObtainer{} }

// acmeUser implements registration.User with an ephemeral account key.
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

	// Ephemeral account key — the dry-run registers a throwaway account.
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate account key: %w", err)
	}
	user := &acmeUser{email: p.Email, key: accountKey}

	cfg := lego.NewConfig(user)
	cfg.CADirURL = p.DirectoryURL

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("acme client: %w", err)
	}

	if err := setSolver(client, p); err != nil {
		return nil, err
	}

	reg, err := client.Registration.Register(ctx, registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("register account: %w", err)
	}
	user.reg = reg

	res, err := client.Certificate.Obtain(ctx, certificate.ObtainRequest{
		Domains: p.Identifiers,
		KeyType: kt,
		Profile: p.Profile,
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("obtain certificate: %w", err)
	}

	// Dry-run: the cert material in res is intentionally discarded (not persisted).
	return &Result{Identifiers: res.Domains}, nil
}

// setSolver configures the challenge solver for the requested challenge type.
func setSolver(client *lego.Client, p Params) error {
	switch p.Challenge {
	case "dns-01":
		provider, err := dns.NewDNSChallengeProviderByName(p.DNSProvider)
		if err != nil {
			return fmt.Errorf("dns provider %q: %w", p.DNSProvider, err)
		}
		return client.Challenge.SetDNS01Provider(provider)
	case "http-01":
		return client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	case "tls-alpn-01":
		return client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", "443"))
	case "dns-persist-01":
		return fmt.Errorf("challenge %q is not wired into issuance yet", p.Challenge)
	default:
		return fmt.Errorf("unsupported challenge %q", p.Challenge)
	}
}
