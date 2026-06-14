package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// accountKey loads — or, on first use, creates and persists — the ACME account
// key for a given directory URL, under base/<hash>/account.key (ADR-0023). One
// reused account per CA, which is what makes revocation possible.
func accountKey(base, directoryURL string) (crypto.Signer, error) {
	dir := accountDir(base, directoryURL)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create account dir: %w", err)
	}
	path := filepath.Join(dir, "account.key")

	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- account-key path under the syscert-owned store, not user input
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("account key %s is not PEM", path)
		}
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse account key %s: %w", path, err)
		}
		return key, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write account key %s: %w", path, err)
	}
	return key, nil
}

// accountDir is the per-CA account directory: base/<first 16 hex of sha256(directoryURL)>.
func accountDir(base, directoryURL string) string {
	sum := sha256.Sum256([]byte(directoryURL))
	return filepath.Join(base, hex.EncodeToString(sum[:8]))
}
