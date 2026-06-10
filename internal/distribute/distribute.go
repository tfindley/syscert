// Package distribute copies store artifacts out to consumer targets with the
// ownership, mode, and (optional, auto-detected) SELinux context each needs.
package distribute

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/tfindley/syscert/internal/atomicfile"
	"github.com/tfindley/syscert/internal/config"
)

// artifactFiles maps a target's artifact name to its store filename.
var artifactFiles = map[string]string{
	"cert":      "cert.pem",
	"privkey":   "privkey.pem",
	"chain":     "chain.pem",
	"fullchain": "fullchain.pem",
	"bundle":    "bundle.pem",
}

// Distributor places artifacts from a store directory onto targets. The system
// effects (chown, SELinux detection/relabel, name lookups) are fields so they
// can be substituted in tests; New wires the production implementations.
type Distributor struct {
	StoreDir       string
	SELinuxEnabled func() bool
	Chown          func(path string, uid, gid int) error
	Relabel        func(path, context string) error
	lookupUID      func(name string) (int, error)
	lookupGID      func(name string) (int, error)
}

// New returns a Distributor wired with production implementations.
func New(storeDir string) *Distributor {
	return &Distributor{
		StoreDir:       storeDir,
		SELinuxEnabled: selinuxEnabled,
		Chown:          os.Chown,
		Relabel:        relabel,
		lookupUID:      lookupUID,
		lookupGID:      lookupGID,
	}
}

// Run places every target's artifact. It writes each file atomically with its
// mode + ownership; if SELinux is active and the target has a context, it
// relabels the file. SELinux is skipped entirely on hosts without it.
func (d *Distributor) Run(targets []config.DistributeTarget) error {
	for _, t := range targets {
		if err := d.one(t); err != nil {
			return err
		}
	}
	return nil
}

func (d *Distributor) one(t config.DistributeTarget) error {
	file, ok := artifactFiles[t.Artifact]
	if !ok {
		return fmt.Errorf("distribute %s: unknown artifact %q", t.Path, t.Artifact)
	}
	data, err := os.ReadFile(filepath.Join(d.StoreDir, file))
	if err != nil {
		return fmt.Errorf("distribute %s: read %s: %w", t.Path, file, err)
	}

	mode := defaultMode(t.Artifact)
	if t.Mode != "" {
		m, err := strconv.ParseUint(t.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("distribute %s: invalid mode %q: %w", t.Path, t.Mode, err)
		}
		mode = os.FileMode(m)
	}

	uid, gid := -1, -1
	if t.Owner != "" {
		if uid, err = d.lookupUID(t.Owner); err != nil {
			return fmt.Errorf("distribute %s: owner %q: %w", t.Path, t.Owner, err)
		}
	}
	if t.Group != "" {
		if gid, err = d.lookupGID(t.Group); err != nil {
			return fmt.Errorf("distribute %s: group %q: %w", t.Path, t.Group, err)
		}
	}

	if err := d.writeAtomic(t.Path, data, mode, uid, gid); err != nil {
		return err
	}

	if t.SELinuxContext != "" && d.SELinuxEnabled() {
		if err := d.Relabel(t.Path, t.SELinuxContext); err != nil {
			return fmt.Errorf("distribute %s: set SELinux context %q: %w", t.Path, t.SELinuxContext, err)
		}
	}
	return nil
}

// writeAtomic writes data to the target path atomically, applying mode and
// (if requested) ownership before the file appears in place.
func (d *Distributor) writeAtomic(path string, data []byte, mode os.FileMode, uid, gid int) error {
	var chown func(string) error
	if uid >= 0 || gid >= 0 {
		chown = func(tmp string) error {
			if err := d.Chown(tmp, uid, gid); err != nil {
				return fmt.Errorf("chown to %d:%d (need CAP_CHOWN or root): %w", uid, gid, err)
			}
			return nil
		}
	}
	if err := atomicfile.Write(path, data, mode, chown); err != nil {
		return fmt.Errorf("distribute %s: %w", path, err)
	}
	return nil
}

func defaultMode(artifact string) os.FileMode {
	if artifact == "privkey" || artifact == "bundle" {
		return 0o600 // key-bearing
	}
	return 0o644
}

// --- production system hooks ---

// selinuxEnabled reports whether SELinux is active (selinuxfs mounted).
func selinuxEnabled() bool {
	_, err := os.Stat("/sys/fs/selinux/enforce")
	return err == nil
}

// relabel sets the SELinux type on path. Only invoked on SELinux hosts.
func relabel(path, context string) error {
	return exec.Command("chcon", "-t", context, path).Run()
}

func lookupUID(name string) (int, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Uid)
}

func lookupGID(name string) (int, error) {
	g, err := user.LookupGroup(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(g.Gid)
}
