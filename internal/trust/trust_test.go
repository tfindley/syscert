package trust

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

func caPEM(t *testing.T, cn string, isCA bool) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestDetectAnchor(t *testing.T) {
	rhel := func(p string) bool { return p == "/etc/pki/ca-trust/source/anchors" }
	dir, cmd, err := detectAnchor(rhel)
	if err != nil || dir != "/etc/pki/ca-trust/source/anchors" || cmd[0] != "update-ca-trust" {
		t.Errorf("RHEL: dir=%q cmd=%v err=%v", dir, cmd, err)
	}

	deb := func(p string) bool { return p == "/usr/local/share/ca-certificates" }
	dir, cmd, err = detectAnchor(deb)
	if err != nil || dir != "/usr/local/share/ca-certificates" || cmd[0] != "update-ca-certificates" {
		t.Errorf("Debian: dir=%q cmd=%v err=%v", dir, cmd, err)
	}

	none := func(string) bool { return false }
	if _, _, err := detectAnchor(none); err == nil {
		t.Error("no anchor dir: want error")
	}
}

func TestInstallWritesAndUpdates(t *testing.T) {
	dir := t.TempDir()
	updates := 0
	m := &Manager{AnchorDir: dir, RunUpdate: func() error { updates++; return nil }}

	pem := caPEM(t, "Test Vault CA", true)
	if err := m.Install("test-vault-ca", pem); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "syscert-test-vault-ca.crt"))
	if err != nil {
		t.Fatalf("anchor not written: %v", err)
	}
	if string(got) != string(pem) {
		t.Error("anchor content mismatch")
	}
	if updates != 1 {
		t.Errorf("RunUpdate called %d times, want 1", updates)
	}
}

func TestRemoveManaged(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"syscert-a.crt", "syscert-b.crt", "other-ca.crt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	updates := 0
	m := &Manager{AnchorDir: dir, RunUpdate: func() error { updates++; return nil }}

	n, err := m.RemoveManaged()
	if err != nil {
		t.Fatalf("RemoveManaged: %v", err)
	}
	if n != 2 {
		t.Errorf("removed %d, want 2", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "other-ca.crt")); err != nil {
		t.Error("non-syscert anchor must be left alone")
	}
	if updates != 1 {
		t.Errorf("RunUpdate called %d times, want 1", updates)
	}
}

func TestCAName(t *testing.T) {
	name, err := CAName(caPEM(t, "Test Vault CA", true))
	if err != nil {
		t.Fatalf("CAName: %v", err)
	}
	if name != "test-vault-ca" {
		t.Errorf("CAName = %q, want test-vault-ca", name)
	}

	if _, err := CAName(caPEM(t, "Leaf", false)); err == nil {
		t.Error("non-CA cert: want error")
	}
	if _, err := CAName([]byte("not pem")); err == nil {
		t.Error("non-PEM: want error")
	}
}
