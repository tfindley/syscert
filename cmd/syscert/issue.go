package main

import (
	"fmt"
	"io"

	"github.com/tfindley/syscert/internal/acme"
)

// cmdIssue obtains a fresh certificate and writes the five artifacts to the
// store. It does NOT distribute — run `syscert distribute` (or bare `syscert`)
// to push to consumers. With --staging it uses the CA's staging directory.
func cmdIssue(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("issue", stderr)
	cfgPath := configFlag(fs)
	envPaths := envFileFlag(fs)
	staging := fs.Bool("staging", false, "use the CA staging directory (Let's Encrypt) — for testing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if code := loadEnvFiles(*envPaths, stderr); code != 0 {
		return code
	}

	cfg, subject, problems, err := loadAndCheck(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "issue: load config: %v\n", err)
		return 2
	}
	if len(problems) > 0 {
		printProblems(stdout, problems)
		return 1
	}
	if !storeAccessGuard("issue", cfg.Store.Path, stderr) {
		return 1
	}

	fmt.Fprintf(stdout, "issuing %s via %s (challenge %s)\n",
		subject, acme.DirectoryURL(cfg, *staging), cfg.EffectiveChallenge())
	if *staging {
		fmt.Fprintln(stdout, "(staging directory — certificate will not be publicly trusted)")
	}
	noteConnectionTrust(cfg, stdout)

	arts, err := provision(cfg, subject, *staging)
	if err != nil {
		fmt.Fprintf(stdout, "FAIL: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "OK: wrote %d artifacts to %s (not distributed — run `syscert distribute`)\n",
		len(arts), cfg.Store.Path)
	return 0
}
