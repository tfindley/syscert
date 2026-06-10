package distribute

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tfindley/syscert/internal/config"
)

type chownCall struct {
	uid, gid int
}
type relabelCall struct {
	path, ctx string
}
type spies struct {
	selinuxOn bool
	chowns    []chownCall
	relabels  []relabelCall
}

// newTest builds a Distributor whose system effects are spied, so tests run
// without privilege or a real SELinux host.
func newTest(store string, sp *spies) *Distributor {
	return &Distributor{
		StoreDir:       store,
		SELinuxEnabled: func() bool { return sp.selinuxOn },
		Chown:          func(_ string, uid, gid int) error { sp.chowns = append(sp.chowns, chownCall{uid, gid}); return nil },
		Relabel:        func(path, ctx string) error { sp.relabels = append(sp.relabels, relabelCall{path, ctx}); return nil },
		lookupUID:      func(string) (int, error) { return 4242, nil },
		lookupGID:      func(string) (int, error) { return 4343, nil },
	}
}

func storeWith(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunWritesArtifactWithModeAndChown(t *testing.T) {
	store := storeWith(t, map[string]string{"fullchain.pem": "FULL"})
	out := filepath.Join(t.TempDir(), "nginx.pem")
	sp := &spies{}
	d := newTest(store, sp)

	err := d.Run([]config.DistributeTarget{{
		Artifact: "fullchain", Path: out, Owner: "root", Group: "ssl", Mode: "0640",
	}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(out)
	if string(data) != "FULL" {
		t.Errorf("content = %q, want FULL", data)
	}
	fi, _ := os.Stat(out)
	if fi.Mode().Perm() != 0o640 {
		t.Errorf("mode = %#o, want 0640", fi.Mode().Perm())
	}
	if len(sp.chowns) != 1 || sp.chowns[0].uid != 4242 || sp.chowns[0].gid != 4343 {
		t.Errorf("chowns = %+v, want one call uid=4242 gid=4343", sp.chowns)
	}
}

func TestRunNoChownWhenOwnerGroupEmpty(t *testing.T) {
	store := storeWith(t, map[string]string{"privkey.pem": "KEY"})
	out := filepath.Join(t.TempDir(), "key.pem")
	sp := &spies{}
	d := newTest(store, sp)

	if err := d.Run([]config.DistributeTarget{{Artifact: "privkey", Path: out}}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(sp.chowns) != 0 {
		t.Errorf("expected no chown when owner/group empty, got %+v", sp.chowns)
	}
	fi, _ := os.Stat(out)
	if fi.Mode().Perm() != 0o600 { // key-bearing default
		t.Errorf("privkey default mode = %#o, want 0600", fi.Mode().Perm())
	}
}

func TestRunUnknownArtifact(t *testing.T) {
	d := newTest(t.TempDir(), &spies{})
	if err := d.Run([]config.DistributeTarget{{Artifact: "bogus", Path: filepath.Join(t.TempDir(), "x")}}); err == nil {
		t.Fatal("unknown artifact: want error")
	}
}

func TestRunMissingArtifactFile(t *testing.T) {
	d := newTest(t.TempDir(), &spies{}) // empty store
	if err := d.Run([]config.DistributeTarget{{Artifact: "cert", Path: filepath.Join(t.TempDir(), "x")}}); err == nil {
		t.Fatal("missing artifact file: want error")
	}
}

func TestRunSELinuxSkippedWhenInactive(t *testing.T) {
	store := storeWith(t, map[string]string{"cert.pem": "C"})
	sp := &spies{selinuxOn: false}
	d := newTest(store, sp)
	d.Run([]config.DistributeTarget{{Artifact: "cert", Path: filepath.Join(t.TempDir(), "c.pem"), SELinuxContext: "cert_t"}})
	if len(sp.relabels) != 0 {
		t.Errorf("relabel must be skipped when SELinux inactive, got %+v", sp.relabels)
	}
}

func TestRunSELinuxSkippedWhenNoContext(t *testing.T) {
	store := storeWith(t, map[string]string{"cert.pem": "C"})
	sp := &spies{selinuxOn: true}
	d := newTest(store, sp)
	d.Run([]config.DistributeTarget{{Artifact: "cert", Path: filepath.Join(t.TempDir(), "c.pem")}}) // no context
	if len(sp.relabels) != 0 {
		t.Errorf("relabel must be skipped when no context set, got %+v", sp.relabels)
	}
}

func TestRunRelabelsWhenActiveAndContextSet(t *testing.T) {
	store := storeWith(t, map[string]string{"cert.pem": "C"})
	out := filepath.Join(t.TempDir(), "c.pem")
	sp := &spies{selinuxOn: true}
	d := newTest(store, sp)
	d.Run([]config.DistributeTarget{{Artifact: "cert", Path: out, SELinuxContext: "cert_t"}})
	if len(sp.relabels) != 1 || sp.relabels[0].ctx != "cert_t" || sp.relabels[0].path != out {
		t.Errorf("relabels = %+v, want one call ctx=cert_t path=%s", sp.relabels, out)
	}
}
