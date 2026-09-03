package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"

	"github.com/tfindley/syscert/internal/config"
)

// targetAccessProblem decides, from an already-attempted write probe, whether a
// distribute target's directory will refuse the real delivery — and says how to
// fix it. Pure, so the wording is unit-tested without privilege or a sandbox.
//
//   - EROFS  → the service sandbox (ProtectSystem=strict) has not granted the
//     directory; the drop-in is generated from the config.
//   - EACCES → ordinary directory permissions; creating a file needs write on the
//     directory, which CAP_CHOWN does not give.
//   - ENOENT → the directory does not exist; syscert writes files, not trees.
//
// probeErr nil means the directory accepted a file, so there is no problem.
func targetAccessProblem(dir string, probeErr error, runAs string) error {
	switch {
	case probeErr == nil:
		return nil
	case errors.Is(probeErr, syscall.EROFS):
		return fmt.Errorf("%s is read-only under the service sandbox (ProtectSystem=strict); "+
			"grant it with 'sudo syscert systemd-paths --write' then 'sudo systemctl daemon-reload'", dir)
	case errors.Is(probeErr, fs.ErrNotExist):
		return fmt.Errorf("%s does not exist; create it before distributing there", dir)
	case errors.Is(probeErr, fs.ErrPermission):
		return fmt.Errorf("%s is not writable by user %s; grant it with 'sudo setfacl -m u:%s:rwx %s'",
			dir, runAs, runAs, dir)
	default:
		return fmt.Errorf("%s is not writable: %v", dir, probeErr)
	}
}

// probeDir reports whether a file can actually be created in dir, by creating
// one and removing it. It deliberately performs the same operation as the real
// delivery (see internal/atomicfile) rather than inspecting mode bits, so it
// cannot disagree with it — mode bits alone miss both the sandbox and ACLs.
func probeDir(dir string) error {
	f, err := os.CreateTemp(dir, ".syscert-probe-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// checkDistributeTargets probes every directory syscert must write outside the
// store — distribute targets and any [observe] output — and returns one problem
// per unwritable directory. An empty result means writing will work *in the
// current context*; see warnDistributeTargets for why that caveat matters.
func checkDistributeTargets(cfg *config.Config) []error {
	runAs := ownerName(os.Geteuid())
	var problems []error
	for _, dir := range outputDirs(cfg) {
		if err := targetAccessProblem(dir, probeDir(dir), runAs); err != nil {
			problems = append(problems, err)
		}
	}
	return problems
}

// reportDistributeTargets is dry-run's view: it lists each target directory and
// whether a file can be created there *right now*, and always prints the drop-in
// the sandboxed service needs. The drop-in is printed unconditionally on purpose
// — dry-run is usually run interactively, where nothing is sandboxed, so a probe
// that passes here says nothing about what the timer will be allowed to do.
func reportDistributeTargets(cfg *config.Config, stdout io.Writer) {
	dirs := outputDirs(cfg)
	if len(dirs) == 0 {
		return
	}
	runAs := ownerName(os.Geteuid())
	fmt.Fprintln(stdout, "\nwritable directories needed:")
	for _, dir := range dirs {
		if err := targetAccessProblem(dir, probeDir(dir), runAs); err != nil {
			fmt.Fprintf(stdout, "  BLOCKED  %s\n           %v\n", dir, err)
			continue
		}
		fmt.Fprintf(stdout, "  writable %s\n", dir)
	}
	fmt.Fprintf(stdout, "\nUnder the systemd timer the service is sandboxed (ProtectSystem=strict) and can\n"+
		"write only what the unit grants, whatever this check saw. Install the grant with\n"+
		"'sudo syscert systemd-paths --write', or add by hand:\n\n%s",
		indent(dropInContent(distributeDirs(cfg)), "  "))
}

// indent prefixes every non-empty line of s with pad.
func indent(s, pad string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// warnDistributeTargets reports unwritable target directories up front and
// returns how many it found. It deliberately WARNS rather than refusing the run.
//
// Refusing would be worse than the problem: ensure would stop before renewing,
// so one unwritable directory would let the certificate expire and break every
// consumer, instead of breaking the one target that is misconfigured. Delivery
// attempts each target independently, so the healthy ones are still served, and
// the run still exits non-zero — the operator is told, loudly, twice.
//
// The probe also reflects the context it runs in: interactively there is no
// sandbox, so a directory the timer would be refused looks writable here. That
// asymmetry is another reason this informs rather than gates.
func warnDistributeTargets(cmd string, cfg *config.Config, stderr io.Writer) int {
	problems := checkDistributeTargets(cfg)
	for _, p := range problems {
		fmt.Fprintf(stderr, "%s: warning: %v\n", cmd, p)
	}
	return len(problems)
}
