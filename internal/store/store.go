// Package store turns ACME cert material into the five on-disk artifacts
// (cert/privkey/chain/fullchain/bundle) and writes them to the canonical store.
package store

import (
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/tfindley/syscert/internal/atomicfile"
)

// certFile is the leaf-certificate artifact name, the source of truth for the
// store layout used across Build, Wipe, and ReadCurrentCert.
const certFile = "cert.pem"

// Material is the certificate material returned by the CA.
type Material struct {
	Certificate       []byte // PEM: leaf + intermediates (the fullchain)
	PrivateKey        []byte // PEM private key
	IssuerCertificate []byte // PEM intermediate(s); optional
	Root              []byte // PEM root; only available from internal CAs
}

// Artifact is one output file: its name, content, and mode.
type Artifact struct {
	Name string
	Data []byte
	Mode os.FileMode
}

// Build produces the five PEM artifacts from cert material (ADR-0030). The leaf
// and intermediate chain are split out of m.Certificate; bundle.pem is assembled
// from the components named in bundleOrder (absent components — e.g. a root that
// the CA didn't provide — are skipped). Key-bearing files are mode 0600.
func Build(m Material, bundleOrder []string) ([]Artifact, error) {
	leaf, chain, err := splitLeafChain(m.Certificate)
	if err != nil {
		return nil, err
	}

	// fullchain is leaf + chain, re-encoded — kept byte-consistent with cert.pem
	// + chain.pem rather than passing the CA's raw bytes through.
	fullchain := make([]byte, 0, len(leaf)+len(chain))
	fullchain = append(fullchain, leaf...)
	fullchain = append(fullchain, chain...)

	bundle := assembleBundle(bundleOrder, leaf, chain, m.Root, m.PrivateKey)
	bundleMode := os.FileMode(0o644)
	if slices.Contains(bundleOrder, "key") {
		bundleMode = 0o600
	}

	return []Artifact{
		{Name: certFile, Data: leaf, Mode: 0o644},
		{Name: "privkey.pem", Data: m.PrivateKey, Mode: 0o600},
		{Name: "chain.pem", Data: chain, Mode: 0o644},
		{Name: "fullchain.pem", Data: fullchain, Mode: 0o644},
		{Name: "bundle.pem", Data: bundle, Mode: bundleMode},
	}, nil
}

// splitLeafChain separates a PEM bundle into its first certificate (leaf) and
// the remaining certificates (intermediate chain), each re-encoded as PEM.
func splitLeafChain(fullchain []byte) (leaf, chain []byte, err error) {
	rest := fullchain
	var blocks []*pem.Block
	for {
		var b *pem.Block
		b, rest = pem.Decode(rest)
		if b == nil {
			break
		}
		blocks = append(blocks, b)
	}
	if len(blocks) == 0 {
		return nil, nil, fmt.Errorf("no PEM certificate blocks in certificate material")
	}
	leaf = pem.EncodeToMemory(blocks[0])
	for _, b := range blocks[1:] {
		chain = append(chain, pem.EncodeToMemory(b)...)
	}
	return leaf, chain, nil
}

// assembleBundle concatenates the requested components in order; nil components
// (e.g. an unavailable root) contribute nothing.
func assembleBundle(order []string, leaf, chain, root, key []byte) []byte {
	var out []byte
	for _, tok := range order {
		switch tok {
		case "cert":
			out = append(out, leaf...)
		case "chain":
			out = append(out, chain...)
		case "root":
			out = append(out, root...)
		case "key":
			out = append(out, key...)
		}
	}
	return out
}

// Write creates dir (if needed) and writes each artifact atomically with its
// mode (temp file + rename), so a reader never sees a partial file (ADR-0025).
func Write(dir string, arts []Artifact) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create store dir %s: %w", dir, err)
	}
	for _, a := range arts {
		if err := atomicfile.Write(filepath.Join(dir, a.Name), a.Data, a.Mode, nil); err != nil {
			return fmt.Errorf("write %s: %w", a.Name, err)
		}
	}
	return nil
}

// Wipe removes the issued artifacts and the ACME account state from the store,
// leaving the directory itself and any unrelated files intact. Returns the
// number of items removed. Used by `destroy` (does not revoke — that's `void`).
func Wipe(dir string) (int, error) {
	removed := 0
	for _, name := range []string{certFile, "privkey.pem", "chain.pem", "fullchain.pem", "bundle.pem"} {
		switch err := os.Remove(filepath.Join(dir, name)); {
		case err == nil:
			removed++
		case !os.IsNotExist(err):
			return removed, fmt.Errorf("remove %s: %w", name, err)
		}
	}
	accounts := filepath.Join(dir, "accounts")
	if _, err := os.Stat(accounts); err == nil {
		if err := os.RemoveAll(accounts); err != nil {
			return removed, fmt.Errorf("remove accounts: %w", err)
		}
		removed++
	}
	return removed, nil
}

// ReadCurrentCert reads the stored leaf certificate (cert.pem) from dir.
func ReadCurrentCert(dir string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, certFile))
}
