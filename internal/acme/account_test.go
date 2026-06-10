package acme

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"testing"
)

func TestAccountKeyPersistsAndReloads(t *testing.T) {
	base := t.TempDir()
	url := "https://vault.example.com/v1/pki/acme/directory"

	k1, err := accountKey(base, url)
	if err != nil {
		t.Fatalf("accountKey (create): %v", err)
	}
	k2, err := accountKey(base, url)
	if err != nil {
		t.Fatalf("accountKey (reload): %v", err)
	}

	pub1 := k1.Public().(*ecdsa.PublicKey)
	if !pub1.Equal(k2.Public()) {
		t.Error("second accountKey returned a different key — not persisted/reused")
	}

	keyPath := filepath.Join(accountDir(base, url), "account.key")
	fi, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("account.key not written: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("account.key mode = %#o, want 0600", fi.Mode().Perm())
	}
}

func TestAccountDirDiffersPerDirectory(t *testing.T) {
	base := t.TempDir()
	if accountDir(base, "https://a.example/dir") == accountDir(base, "https://b.example/dir") {
		t.Error("different directory URLs must map to different account dirs")
	}
}
