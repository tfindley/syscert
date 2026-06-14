// Package atomicfile writes a file atomically: contents go to a temp file in
// the same directory, which is then renamed into place — so a reader (or a
// watcher) never observes a partial or wrong-permissioned file (ADR-0025).
package atomicfile

import (
	"os"
	"path/filepath"
)

// Write writes data to path atomically with the given mode. If prepare is
// non-nil it runs against the temp file just before the rename (e.g. to chown),
// so ownership/labels are applied before the file appears at its final path.
func Write(path string, data []byte, mode os.FileMode, prepare func(tmpPath string) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".syscert-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // harmless no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if prepare != nil {
		if err := prepare(tmpName); err != nil {
			return err
		}
	}
	return os.Rename(tmpName, path)
}
