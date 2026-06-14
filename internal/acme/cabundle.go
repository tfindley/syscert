package acme

import (
	"crypto/x509"
	"fmt"
	"os"
)

// loadCABundle reads a PEM file of CA certificate(s) and returns a pool trusting
// them. Used to trust an internal CA for the ACME connection only (ADR-0035),
// without touching the system trust store.
func loadCABundle(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path) //nosec G304 -- operator-configured acme.ca_bundle path
	if err != nil {
		return nil, fmt.Errorf("read ca_bundle %s: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("ca_bundle %s contains no usable PEM certificates", path)
	}
	return pool, nil
}
