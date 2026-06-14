// Package trust manages the internal-CA chain in the system trust store
// (ADR-0011): a root-only operation, distinct from the connection-only trust of
// acme.ca_bundle (ADR-0035).
package trust

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const anchorPrefix = "syscert-"

// Manager installs/removes CA anchors in the platform trust store.
type Manager struct {
	AnchorDir string       // where anchor .crt files live
	RunUpdate func() error // refreshes the system trust store
}

// New detects the platform's trust-anchor directory + update command and wires
// the production implementation.
func New() (*Manager, error) {
	dir, cmd, err := detectAnchor(dirExists)
	if err != nil {
		return nil, err
	}
	return &Manager{
		AnchorDir: dir,
		RunUpdate: func() error {
			if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil { // #nosec G204 -- cmd is the OS-detected trust-store update command, not user input
				return fmt.Errorf("%s: %w: %s", strings.Join(cmd, " "), err, out)
			}
			return nil
		},
	}, nil
}

// Install writes the CA bundle as a SysCert-managed anchor and refreshes the
// trust store. Idempotent (overwrites an existing anchor of the same name).
func (m *Manager) Install(name string, caPEM []byte) error {
	path := filepath.Join(m.AnchorDir, anchorPrefix+name+".crt")
	if err := os.WriteFile(path, caPEM, 0o644); err != nil { // #nosec G306 -- CA anchors are public certs; must be world-readable in the trust store
		return fmt.Errorf("write anchor %s: %w", path, err)
	}
	return m.RunUpdate()
}

// RemoveManaged deletes every SysCert-managed anchor (prefix "syscert-") and
// refreshes the trust store. Returns how many were removed.
func (m *Manager) RemoveManaged() (int, error) {
	matches, err := filepath.Glob(filepath.Join(m.AnchorDir, anchorPrefix+"*.crt"))
	if err != nil {
		return 0, err
	}
	for _, p := range matches {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return 0, fmt.Errorf("remove anchor %s: %w", p, err)
		}
	}
	if len(matches) == 0 {
		return 0, nil
	}
	return len(matches), m.RunUpdate()
}

func detectAnchor(exists func(string) bool) (dir string, cmd []string, err error) {
	switch {
	case exists("/etc/pki/ca-trust/source/anchors"): // RHEL family
		return "/etc/pki/ca-trust/source/anchors", []string{"update-ca-trust", "extract"}, nil
	case exists("/usr/local/share/ca-certificates"): // Debian/Ubuntu
		return "/usr/local/share/ca-certificates", []string{"update-ca-certificates"}, nil
	}
	return "", nil, fmt.Errorf("no supported CA trust anchor directory found (need Debian/Ubuntu or RHEL family)")
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// CAName returns a filesystem-safe name derived from the first CA certificate's
// common name in the bundle. Errors if no CA certificate is present.
func CAName(caPEM []byte) (string, error) {
	rest := caPEM
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !cert.IsCA {
			continue
		}
		if name := sanitize(cert.Subject.CommonName); name != "" {
			return name, nil
		}
		return "ca", nil
	}
	return "", fmt.Errorf("no CA certificate found in bundle")
}

// sanitize lowercases and replaces runs of non-alphanumeric characters with a
// single hyphen, suitable for an anchor filename.
func sanitize(s string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			b.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}
