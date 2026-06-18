package main

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// storeAccessProblem decides, from already-resolved facts, whether a write
// command must refuse before touching the store, returning the refusal error
// (or nil to proceed). It is pure so the decision is unit-tested without a real
// filesystem or privilege.
//
//   - root (euid 0) over a store owned by a non-root user → refuse: files root
//     creates would be unreadable/unrenewable by that user (issue #2).
//   - a non-root user whose euid is not the store owner → refuse with a clean
//     message instead of a raw mkdir/permission-denied later (issue #3).
//   - owner running over its own store, or root over a root-owned store → nil.
//
// ownerName is used verbatim in the message; callers pass the numeric uid as a
// string when name lookup fails.
func storeAccessProblem(storePath string, ownerUID, euid int, ownerName string) error {
	if euid == 0 {
		if ownerUID != 0 {
			return fmt.Errorf("store %s is owned by %s; running as root would create files %s can't renew — run as that user: sudo -u %s syscert …",
				storePath, ownerName, ownerName, ownerName)
		}
		return nil
	}
	if euid != ownerUID {
		return fmt.Errorf("store %s is owned by %s; run syscert as that user or root (the systemd timer does this for you)",
			storePath, ownerName)
	}
	return nil
}

// checkStoreAccess is the thin wrapper the write commands run before touching
// the store. It stats storePath, reads the owning uid, resolves it to a name
// (falling back to the numeric uid), and applies storeAccessProblem against the
// current euid. A store that does not exist yet is a no-op (normal creation
// proceeds); a stat error other than not-exist is surfaced.
func checkStoreAccess(storePath string) error {
	fi, err := os.Stat(storePath)
	if os.IsNotExist(err) {
		return nil // not created yet — let issuance create it
	}
	if err != nil {
		return fmt.Errorf("stat store %s: %w", storePath, err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil // non-Unix stat backing — can't determine ownership, don't block
	}
	ownerUID := int(st.Uid)
	return storeAccessProblem(storePath, ownerUID, os.Geteuid(), ownerName(ownerUID))
}

// ownerName resolves a uid to its login name, falling back to the numeric uid
// (as a string) when lookup fails — so the refusal message is always concrete.
func ownerName(uid int) string {
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
		return u.Username
	}
	return strconv.Itoa(uid)
}
