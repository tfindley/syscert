package renewal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func makeCertPEM(t *testing.T, notBefore, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     []string{"host.example.com"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestDueAutoWindowThirdOfLife(t *testing.T) {
	now := time.Now()
	// 90-day cert: auto window = 30d.
	cert := makeCertPEM(t, now.Add(-80*24*time.Hour), now.Add(10*24*time.Hour)) // 10 days left → due
	due, err := Due(cert, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Error("10 days left on a 90-day cert (window 30d): want due=true")
	}

	cert2 := makeCertPEM(t, now.Add(-1*24*time.Hour), now.Add(89*24*time.Hour)) // 89 days left → not due
	due2, _ := Due(cert2, "", now)
	if due2 {
		t.Error("89 days left on a 90-day cert (window 30d): want due=false")
	}
}

func TestDueShortLivedAutoWindow(t *testing.T) {
	now := time.Now()
	// 6-day cert: auto window = 2 days.
	due, _ := Due(makeCertPEM(t, now.Add(-5*24*time.Hour), now.Add(1*24*time.Hour)), "", now)
	if !due {
		t.Error("1 day left on a 6-day cert (window 2d): want due=true")
	}
	notDue, _ := Due(makeCertPEM(t, now.Add(-2*24*time.Hour), now.Add(4*24*time.Hour)), "", now)
	if notDue {
		t.Error("4 days left on a 6-day cert (window 2d): want due=false")
	}
}

func TestDueExplicitRenewBefore(t *testing.T) {
	now := time.Now()
	cert := makeCertPEM(t, now.Add(-60*24*time.Hour), now.Add(20*24*time.Hour)) // 20 days left
	if due, _ := Due(cert, "30d", now); !due {
		t.Error("20 days left, renew_before=30d: want due=true")
	}
	if due, _ := Due(cert, "10d", now); due {
		t.Error("20 days left, renew_before=10d: want due=false")
	}
}

func TestDueCorruptCert(t *testing.T) {
	if _, err := Due([]byte("not a pem cert"), "", time.Now()); err == nil {
		t.Error("corrupt cert: want error")
	}
}

func TestParseWindow(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"48h", 48 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"", 0, true}, // empty is not a valid explicit window
		{"banana", 0, true},
	}
	for _, c := range cases {
		got, err := parseWindow(c.in)
		if c.err {
			if err == nil {
				t.Errorf("parseWindow(%q): want error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseWindow(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseWindow(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
