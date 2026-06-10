package acme

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCAPEM(t *testing.T, isCA bool) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadCABundleValid(t *testing.T) {
	pool, err := loadCABundle(writeCAPEM(t, true))
	if err != nil {
		t.Fatalf("loadCABundle: %v", err)
	}
	if pool == nil {
		t.Fatal("expected a non-nil cert pool")
	}
}

func TestLoadCABundleMissingFile(t *testing.T) {
	if _, err := loadCABundle("/no/such/ca.pem"); err == nil {
		t.Fatal("missing file: want error")
	}
}

func TestLoadCABundleNotPEM(t *testing.T) {
	p := filepath.Join(t.TempDir(), "junk.pem")
	if err := os.WriteFile(p, []byte("definitely not a pem certificate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCABundle(p); err == nil {
		t.Fatal("non-PEM content: want error")
	}
}
