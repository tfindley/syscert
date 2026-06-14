package acme

import (
	"context"
	"errors"
	"testing"

	"github.com/tfindley/syscert/internal/config"
)

type fakeObtainer struct {
	got         Params
	res         *Result
	err         error
	gotRevoke   Params
	revokedCert []byte
	revokeErr   error
}

func (f *fakeObtainer) Obtain(_ context.Context, p Params) (*Result, error) {
	f.got = p
	return f.res, f.err
}

func (f *fakeObtainer) Revoke(_ context.Context, p Params, certPEM []byte) error {
	f.gotRevoke = p
	f.revokedCert = certPEM
	return f.revokeErr
}

func leConfig() *config.Config {
	return &config.Config{
		Cert: config.CertConfig{KeyType: "ec256"},
		ACME: config.ACMEConfig{CA: "letsencrypt", Email: "a@example.com", Challenge: "dns-01"},
	}
}

func TestDirectoryURLStagingForLEInDryRun(t *testing.T) {
	got := DirectoryURL(leConfig(), true)
	if got != leStagingURL {
		t.Errorf("dry-run LE directory = %q, want staging %q", got, leStagingURL)
	}
}

func TestDirectoryURLProductionForLEOutsideDryRun(t *testing.T) {
	got := DirectoryURL(leConfig(), false)
	if got != leProductionURL {
		t.Errorf("non-dry-run LE directory = %q, want production %q", got, leProductionURL)
	}
}

func TestDirectoryURLExplicitWins(t *testing.T) {
	c := leConfig()
	c.ACME.DirectoryURL = "https://acme.example.test/directory"
	if got := DirectoryURL(c, true); got != c.ACME.DirectoryURL {
		t.Errorf("explicit directory_url = %q, want it honoured even in dry-run", got)
	}
}

func TestDirectoryURLInternalCAUsesConfigured(t *testing.T) {
	c := &config.Config{ACME: config.ACMEConfig{CA: "custom", DirectoryURL: "https://vault/v1/pki/acme/directory"}}
	if got := DirectoryURL(c, true); got != "https://vault/v1/pki/acme/directory" {
		t.Errorf("custom directory = %q", got)
	}
}

func TestDNSPropagationOpts(t *testing.T) {
	cases := map[string]int{"": 0, "all": 0, "authoritative": 1, "off": 2}
	for mode, want := range cases {
		if got := len(dnsPropagationOpts(mode)); got != want {
			t.Errorf("dnsPropagationOpts(%q) = %d opts, want %d", mode, got, want)
		}
	}
}

func TestEABOptions(t *testing.T) {
	if _, useEAB, err := eabOptions(Params{}); useEAB || err != nil {
		t.Errorf("no kid: useEAB=%v err=%v, want false/nil", useEAB, err)
	}
	if _, _, err := eabOptions(Params{EABKid: "k"}); err == nil {
		t.Error("kid without hmac: want error, got nil")
	}
	opts, useEAB, err := eabOptions(Params{EABKid: "k", EABHMAC: "h"})
	if err != nil || !useEAB {
		t.Fatalf("kid+hmac: useEAB=%v err=%v", useEAB, err)
	}
	if opts.Kid != "k" || opts.HmacEncoded != "h" || !opts.TermsOfServiceAgreed {
		t.Errorf("opts = %+v, want Kid=k HmacEncoded=h TOS=true", opts)
	}
}

func TestParamsReadsEAB(t *testing.T) {
	t.Setenv(envEABHMAC, "secret-hmac")
	cfg := &config.Config{ACME: config.ACMEConfig{EAB: config.ACMEEABConfig{Kid: "my-kid"}}}
	p := params(cfg, "host.example.com", false, "")
	if p.EABKid != "my-kid" {
		t.Errorf("EABKid = %q, want my-kid", p.EABKid)
	}
	if p.EABHMAC != "secret-hmac" {
		t.Errorf("EABHMAC = %q, want the env value", p.EABHMAC)
	}
}

func TestIdentifiers(t *testing.T) {
	c := &config.Config{Cert: config.CertConfig{
		SANs:   []string{"alt.example.com", "host.example.com"}, // dup of subject
		IPSANs: []string{"10.0.0.5"},
	}}
	got := Identifiers(c, "host.example.com")
	want := []string{"host.example.com", "alt.example.com", "10.0.0.5"}
	if len(got) != len(want) {
		t.Fatalf("Identifiers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Identifiers = %v, want %v", got, want)
		}
	}
}

func TestDryRunPassesStagingAndIdentifiers(t *testing.T) {
	c := leConfig()
	c.Cert.SANs = []string{"alt.example.com"}
	f := &fakeObtainer{res: &Result{Identifiers: []string{"host.example.com"}}}

	if _, err := DryRun(context.Background(), c, "host.example.com", f); err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if f.got.DirectoryURL != leStagingURL {
		t.Errorf("obtainer got directory %q, want staging", f.got.DirectoryURL)
	}
	if len(f.got.Identifiers) != 2 || f.got.Identifiers[0] != "host.example.com" {
		t.Errorf("obtainer got identifiers %v", f.got.Identifiers)
	}
	if f.got.Challenge != "dns-01" {
		t.Errorf("obtainer got challenge %q", f.got.Challenge)
	}
}

func TestDryRunAutoSwitchesChallengeForIPSAN(t *testing.T) {
	// An IP SAN with the default dns-01 must reach the obtainer as http-01.
	c := leConfig()
	c.Cert.IPSANs = []string{"203.0.113.5"}
	f := &fakeObtainer{res: &Result{}}
	if _, err := DryRun(context.Background(), c, "host.example.com", f); err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if f.got.Challenge != "http-01" {
		t.Errorf("obtainer got challenge %q, want http-01 (auto-switched)", f.got.Challenge)
	}
}

func TestObtainProductionDirectoryWhenNotStaging(t *testing.T) {
	f := &fakeObtainer{res: &Result{}}
	if _, err := Obtain(context.Background(), leConfig(), "host.example.com", f, false); err != nil {
		t.Fatalf("Obtain: %v", err)
	}
	if f.got.DirectoryURL != leProductionURL {
		t.Errorf("non-staging Obtain directory = %q, want production %q", f.got.DirectoryURL, leProductionURL)
	}
}

func TestRevokeCertUsesAccountAndCert(t *testing.T) {
	c := leConfig()
	c.Store.Path = "/var/lib/syscert"
	f := &fakeObtainer{}
	if err := RevokeCert(context.Background(), c, []byte("CERTPEM"), f, false); err != nil {
		t.Fatalf("RevokeCert: %v", err)
	}
	if string(f.revokedCert) != "CERTPEM" {
		t.Errorf("revoked cert = %q", f.revokedCert)
	}
	if f.gotRevoke.AccountDir != "/var/lib/syscert/accounts" {
		t.Errorf("revoke AccountDir = %q, want /var/lib/syscert/accounts", f.gotRevoke.AccountDir)
	}
	if f.gotRevoke.DirectoryURL != leProductionURL {
		t.Errorf("revoke directory = %q, want production", f.gotRevoke.DirectoryURL)
	}
}

func TestObtainUsesPersistentAccountDir(t *testing.T) {
	c := leConfig()
	c.Store.Path = "/srv/syscert"
	f := &fakeObtainer{res: &Result{}}
	if _, err := Obtain(context.Background(), c, "host.example.com", f, false); err != nil {
		t.Fatalf("Obtain: %v", err)
	}
	if f.got.AccountDir != "/srv/syscert/accounts" {
		t.Errorf("Obtain AccountDir = %q, want /srv/syscert/accounts", f.got.AccountDir)
	}
}

func TestDryRunUsesEphemeralAccount(t *testing.T) {
	f := &fakeObtainer{res: &Result{}}
	if _, err := DryRun(context.Background(), leConfig(), "host.example.com", f); err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if f.got.AccountDir != "" {
		t.Errorf("DryRun AccountDir = %q, want empty (ephemeral)", f.got.AccountDir)
	}
}

func TestDryRunPropagatesError(t *testing.T) {
	f := &fakeObtainer{err: errors.New("acme: order failed")}
	if _, err := DryRun(context.Background(), leConfig(), "host.example.com", f); err == nil {
		t.Fatal("DryRun: want error from obtainer, got nil")
	}
}
