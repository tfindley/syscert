package main

import (
	"fmt"
	"io"

	"github.com/tfindley/syscert/internal/distribute"
)

// cmdDistribute re-copies the already-stored artifacts to the configured
// targets, without re-issuing — useful after editing distribution targets.
func cmdDistribute(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("distribute", stderr)
	cfgPath := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "distribute: load config: %v\n", err)
		return 2
	}

	if err := distribute.New(cfg.Store.Path).Run(cfg.Distribute); err != nil {
		fmt.Fprintf(stderr, "distribute: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "OK: distributed %d target(s) from %s\n", len(cfg.Distribute), cfg.Store.Path)
	return 0
}
