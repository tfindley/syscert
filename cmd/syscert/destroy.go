package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/store"
	"github.com/tfindley/syscert/internal/trust"
)

// cmdDestroy tears down SysCert state (issued cert + ACME account) so the host
// can be re-provisioned (e.g. switching CA). It does NOT revoke the current cert
// (run `void` for that) and does NOT reissue. Interactive unless --force.
func cmdDestroy(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("destroy", stderr)
	cfgPath := configFlag(fs)
	force := fs.Bool("force", false, "skip the interactive confirmation(s)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "destroy: load config: %v\n", err)
		return 2
	}

	// One reader for the whole command, so buffered/type-ahead input isn't lost
	// between the two prompts.
	in := bufio.NewReader(os.Stdin)

	prompt := fmt.Sprintf("Destroy SysCert state in %s (certificate + ACME account)? "+
		"This does NOT revoke the current cert — run `void` first if you need that. [y/N] ", cfg.Store.Path)
	if !confirm(stdout, in, prompt, *force) {
		fmt.Fprintln(stdout, "aborted.")
		return 0
	}

	removed, err := store.Wipe(cfg.Store.Path)
	if err != nil {
		fmt.Fprintf(stderr, "destroy: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "removed %d item(s) from %s\n", removed, cfg.Store.Path)

	// Offer to un-trust an internal CA's anchors (root). Public CAs have none.
	if config.IsInternalCA(cfg.ACME.CA) {
		untrust := *force ||
			confirm(stdout, in, "Also remove the internal CA from the system trust store (requires root)? [y/N] ", false)
		if untrust {
			if mgr, err := trust.New(); err != nil {
				fmt.Fprintf(stderr, "note: system trust store unavailable: %v\n", err)
			} else if n, err := mgr.RemoveManaged(); err != nil {
				fmt.Fprintf(stderr, "note: could not remove CA anchors (need root?): %v\n", err)
			} else {
				fmt.Fprintf(stdout, "removed %d CA anchor(s) from %s\n", n, mgr.AnchorDir)
			}
		}
	}

	fmt.Fprintln(stdout, "OK: destroyed. Reconfigure and run `syscert` to provision again.")
	return 0
}
