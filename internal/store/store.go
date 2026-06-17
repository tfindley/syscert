// Package store turns ACME cert material into the five on-disk artifacts
// (cert/privkey/chain/fullchain/bundle) and writes them to the canonical store.
package store

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/tfindley/syscert/internal/atomicfile"
)

// certFile is the leaf-certificate artifact name, the source of truth for the
// store layout used across Build, Wipe, and ReadCurrentCert.
const certFile = "cert.pem"

// artifactNames is the canonical set of store files, shared by Write, Archive, and
// Wipe so the layout stays defined in one place.
var artifactNames = []string{certFile, "privkey.pem", "chain.pem", "fullchain.pem", "bundle.pem"}

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

// WriteOptions controls the store directory's permissions. The zero value keeps
// the historical 0700, syscert-owned layout.
type WriteOptions struct {
	DirMode os.FileMode // store-dir mode; 0 → 0o700
	Group   string      // chgrp the store dir + files to this group; "" → leave as-is
}

// Write creates dir with the configured mode (if needed) and writes each artifact
// atomically with its own mode (temp file + rename), so a reader never sees a partial
// file (ADR-0025). Per-file modes come from Build — key-bearing files stay 0600
// regardless of WriteOptions; only the directory mode/group are configurable (ADR-0041).
func Write(dir string, arts []Artifact, opts WriteOptions) error {
	mode := opts.DirMode
	if mode == 0 {
		mode = 0o700
	}
	if err := os.MkdirAll(dir, mode); err != nil {
		return fmt.Errorf("create store dir %s: %w", dir, err)
	}
	if err := os.Chmod(dir, mode); err != nil { // MkdirAll won't re-mode an existing dir
		return fmt.Errorf("chmod store dir %s: %w", dir, err)
	}
	gid := -1
	if opts.Group != "" {
		g, err := lookupGID(opts.Group)
		if err != nil {
			return fmt.Errorf("store group %q: %w", opts.Group, err)
		}
		gid = g
		if err := os.Chown(dir, -1, gid); err != nil {
			return fmt.Errorf("chgrp store dir: %w", err)
		}
	}
	for _, a := range arts {
		p := filepath.Join(dir, a.Name)
		if err := atomicfile.Write(p, a.Data, a.Mode, nil); err != nil {
			return fmt.Errorf("write %s: %w", a.Name, err)
		}
		if gid >= 0 {
			if err := os.Chown(p, -1, gid); err != nil {
				return fmt.Errorf("chgrp %s: %w", a.Name, err)
			}
		}
	}
	return nil
}

// lookupGID resolves a group name to its numeric GID.
func lookupGID(name string) (int, error) {
	g, err := user.LookupGroup(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(g.Gid)
}

// Archive snapshots the current artifacts into dir/archive/<UTC-timestamp>/ before
// they are overwritten, keeping the keep most recent snapshots (keep<=0 disables it;
// a missing current cert is a no-op). Snapshots are real files — no symlinks, unlike
// certbot's live/archive — kept inside the locked store and never distributed, so
// consumers are unaffected (ADR-0040).
func Archive(dir string, keep int) error {
	if keep <= 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, certFile)); err != nil {
		return nil // nothing issued yet — nothing to archive
	}
	dest := filepath.Join(dir, "archive", time.Now().UTC().Format("2006-01-02T15-04-05Z"))
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}
	for _, name := range artifactNames {
		src := filepath.Join(dir, name)
		data, err := os.ReadFile(src) // #nosec G304 -- syscert-owned store path from config, not user input
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s for archive: %w", name, err)
		}
		fi, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("stat %s for archive: %w", name, err)
		}
		if err := atomicfile.Write(filepath.Join(dest, name), data, fi.Mode().Perm(), nil); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}
	return pruneArchive(dir, keep)
}

// pruneArchive keeps the keep newest snapshot directories under dir/archive and
// removes the rest. Snapshot names are UTC timestamps, so lexical order is age order.
func pruneArchive(dir string, keep int) error {
	root := filepath.Join(dir, "archive")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil // no archive dir yet — nothing to prune
	}
	var snaps []string
	for _, e := range entries {
		if e.IsDir() {
			snaps = append(snaps, e.Name())
		}
	}
	sort.Strings(snaps)
	for i := 0; i < len(snaps)-keep; i++ {
		if err := os.RemoveAll(filepath.Join(root, snaps[i])); err != nil {
			return fmt.Errorf("prune archive %s: %w", snaps[i], err)
		}
	}
	return nil
}

// WipeCerts removes the issued artifacts and any archived snapshots, but KEEPS the
// ACME account state (accounts/) — so the next issuance reuses the account and, with
// EAB, needs no fresh token. Returns the number of items removed. Used by
// `destroy --keep-account`.
func WipeCerts(dir string) (int, error) {
	removed := 0
	for _, name := range artifactNames {
		switch err := os.Remove(filepath.Join(dir, name)); {
		case err == nil:
			removed++
		case !os.IsNotExist(err):
			return removed, fmt.Errorf("remove %s: %w", name, err)
		}
	}
	n, err := removeIfPresent(dir, "archive")
	return removed + n, err
}

// Wipe removes the issued artifacts, archived snapshots, AND the ACME account state
// — a full teardown (e.g. switching CA). Returns the number of items removed. Used by
// `destroy` (does not revoke — that's `void`).
func Wipe(dir string) (int, error) {
	removed, err := WipeCerts(dir)
	if err != nil {
		return removed, err
	}
	n, err := removeIfPresent(dir, "accounts")
	return removed + n, err
}

// removeIfPresent removes dir/sub (recursively) if it exists, returning 1 when it did.
func removeIfPresent(dir, sub string) (int, error) {
	p := filepath.Join(dir, sub)
	if _, err := os.Stat(p); err != nil {
		return 0, nil
	}
	if err := os.RemoveAll(p); err != nil {
		return 0, fmt.Errorf("remove %s: %w", sub, err)
	}
	return 1, nil
}

// ReadCurrentCert reads the stored leaf certificate (cert.pem) from dir.
func ReadCurrentCert(dir string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, certFile)) // #nosec G304 -- syscert-owned store path from config, not user input
}

// CountAccounts reports how many per-CA account directories exist under dir/accounts.
func CountAccounts(dir string) int {
	entries, err := os.ReadDir(filepath.Join(dir, "accounts"))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}

// ListArchive returns the archive snapshot names (UTC timestamps) under dir/archive,
// oldest first; nil when there are none.
func ListArchive(dir string) []string {
	entries, err := os.ReadDir(filepath.Join(dir, "archive"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
