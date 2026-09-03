// Package distribute copies store artifacts out to consumer targets with the
// ownership, mode, and (optional, auto-detected) SELinux context each needs.
package distribute

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

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
//
// Every target is attempted even when an earlier one fails, and the failures are
// returned joined. One target the service cannot write — a privileged directory
// that has not been granted, say — must not deny every other consumer its
// renewed certificate. The caller still sees a non-nil error and still exits
// non-zero, so a broken target remains loud.
func (d *Distributor) Run(targets []config.DistributeTarget) error {
	var errs []error
	for _, t := range targets {
		if err := d.one(t); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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
		return fmt.Errorf("distribute %s: %w", path, explainWriteError(filepath.Dir(path), err))
	}
	return nil
}

// explainWriteError turns the raw errno from an atomic write into something the
// operator can act on. Two failures account for essentially every report, and
// both look like an opaque errno without this: the service sandbox
// (ProtectSystem=strict leaves everything outside ReadWritePaths read-only) and
// plain directory permissions (creating a file needs write on the *directory*,
// which CAP_CHOWN does not grant — that only re-owns a file already created).
// Anything else is returned unchanged rather than guessed at.
func explainWriteError(dir string, err error) error {
	switch {
	case errors.Is(err, syscall.EROFS):
		return fmt.Errorf("%s is read-only under the service sandbox (ProtectSystem=strict); "+
			"grant it with 'sudo syscert systemd-paths --write' then 'sudo systemctl daemon-reload': %w", dir, err)
	case errors.Is(err, fs.ErrPermission):
		u := runningUser()
		return fmt.Errorf("%s is not writable by user %s (creating a file needs write on the directory; "+
			"CAP_CHOWN does not grant that); grant it with 'sudo setfacl -m u:%s:rwx %s': %w", dir, u, u, dir, err)
	}
	return err
}

// runningUser names the effective user for an error message, falling back to the
// numeric uid so the message is always concrete.
func runningUser() string {
	uid := os.Geteuid()
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
		return u.Username
	}
	return strconv.Itoa(uid)
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
	return exec.Command("chcon", "-t", context, path).Run() // #nosec G204 -- fixed chcon with operator-configured selinux_context + distribute path
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
