package acme

import (
	"context"
	"errors"
	"testing"

	"github.com/tfindley/syscert/internal/config"
)

type fakeObtainer struct {
	got Params
	res *Result
	err error
}

func (f *fakeObtainer) Obtain(_ context.Context, p Params) (*Result, error) {
	f.got = p
	return f.res, f.err
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
	c := &config.Config{ACME: config.ACMEConfig{CA: "vault", DirectoryURL: "https://vault/v1/pki/acme/directory"}}
	if got := DirectoryURL(c, true); got != "https://vault/v1/pki/acme/directory" {
		t.Errorf("vault directory = %q", got)
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

func TestDryRunPropagatesError(t *testing.T) {
	f := &fakeObtainer{err: errors.New("acme: order failed")}
	if _, err := DryRun(context.Background(), leConfig(), "host.example.com", f); err == nil {
		t.Fatal("DryRun: want error from obtainer, got nil")
	}
}
