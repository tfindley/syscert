package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestStoreAccessProblem pins the store-ownership preflight decisions (issues
// #2 and #3). The function is pure: given the resolved facts it returns the
// error to refuse with, or nil to proceed.
func TestStoreAccessProblem(t *testing.T) {
	const path = "/var/lib/syscert"

	t.Run("root over other-user-owned store is refused (#2)", func(t *testing.T) {
		err := storeAccessProblem(path, 1000, 0, "alice")
		if err == nil {
			t.Fatal("want refusal: root must not write a store owned by a non-root user")
		}
		msg := err.Error()
		for _, want := range []string{path, "alice", "sudo -u alice syscert"} {
			if !strings.Contains(msg, want) {
				t.Errorf("message %q missing %q", msg, want)
			}
		}
	})

	t.Run("unprivileged over other-user store is refused (#3)", func(t *testing.T) {
		err := storeAccessProblem(path, 1000, 1001, "alice")
		if err == nil {
			t.Fatal("want refusal: a non-owner non-root user cannot write the store")
		}
		msg := err.Error()
		for _, want := range []string{path, "alice", "run syscert as that user or root"} {
			if !strings.Contains(msg, want) {
				t.Errorf("message %q missing %q", msg, want)
			}
		}
		// Must not leak the raw mkdir/permission-denied phrasing.
		if strings.Contains(msg, "permission denied") {
			t.Errorf("message %q should not surface raw permission-denied text", msg)
		}
	})

	t.Run("euid equals owner is allowed", func(t *testing.T) {
		if err := storeAccessProblem(path, 1000, 1000, "syscert"); err != nil {
			t.Errorf("owner running over its own store should pass, got %v", err)
		}
	})

	t.Run("root over root-owned store is allowed", func(t *testing.T) {
		if err := storeAccessProblem(path, 0, 0, "root"); err != nil {
			t.Errorf("root over a root-owned store should pass, got %v", err)
		}
	})
}

// TestCheckStoreAccess exercises the stat-based wrapper for the cases reachable
// without elevated privilege: a not-yet-created store (no-op) and a store the
// current user already owns (the test creates it).
func TestCheckStoreAccess(t *testing.T) {
	t.Run("missing store is a no-op", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "not-created-yet")
		if err := checkStoreAccess(missing); err != nil {
			t.Errorf("missing store should pass (creation proceeds), got %v", err)
		}
	})

	t.Run("store owned by the running user passes", func(t *testing.T) {
		dir := t.TempDir() // owned by the test process's euid
		if err := checkStoreAccess(dir); err != nil {
			t.Errorf("self-owned store should pass, got %v", err)
		}
	})
}
