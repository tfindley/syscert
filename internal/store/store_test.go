package store

import (
	"bytes"
	"encoding/pem"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func certPEM(body string) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte(body)})
}

func keyPEM(body string) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte(body)})
}

func sampleMaterial() Material {
	leaf := certPEM("leaf")
	inter := certPEM("inter")
	return Material{
		Certificate: append(append([]byte{}, leaf...), inter...), // fullchain: leaf + intermediate
		PrivateKey:  keyPEM("key"),
		Root:        certPEM("root"),
	}
}

func find(t *testing.T, arts []Artifact, name string) Artifact {
	t.Helper()
	for _, a := range arts {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("artifact %q not produced", name)
	return Artifact{}
}

func TestBuildSplitsArtifacts(t *testing.T) {
	leaf, inter := certPEM("leaf"), certPEM("inter")
	arts, err := Build(sampleMaterial(), []string{"cert", "chain", "root", "key"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := map[string][]byte{
		"cert.pem":      leaf,
		"chain.pem":     inter,
		"fullchain.pem": append(append([]byte{}, leaf...), inter...),
		"privkey.pem":   keyPEM("key"),
		"bundle.pem":    bytes.Join([][]byte{leaf, inter, certPEM("root"), keyPEM("key")}, nil),
	}
	for name, w := range want {
		if got := find(t, arts, name).Data; !bytes.Equal(got, w) {
			t.Errorf("%s\n got: %q\nwant: %q", name, got, w)
		}
	}
}

func TestBuildFullchainEqualsCertPlusChain(t *testing.T) {
	// Even with irregular spacing in the input, fullchain.pem must be exactly
	// cert.pem followed by chain.pem (consistent, normalized PEM).
	leaf, inter := certPEM("leaf"), certPEM("inter")
	m := Material{
		Certificate: bytes.Join([][]byte{leaf, []byte("\n\n"), inter}, nil), // stray blank line
		PrivateKey:  keyPEM("key"),
	}
	arts, err := Build(m, []string{"cert", "chain"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cert := find(t, arts, "cert.pem").Data
	chain := find(t, arts, "chain.pem").Data
	full := find(t, arts, "fullchain.pem").Data
	if want := bytes.Join([][]byte{cert, chain}, nil); !bytes.Equal(full, want) {
		t.Errorf("fullchain != cert+chain\n full: %q\n want: %q", full, want)
	}
}

func TestBuildModes(t *testing.T) {
	arts, _ := Build(sampleMaterial(), []string{"cert", "chain", "root", "key"})
	want := map[string]os.FileMode{
		"cert.pem": 0o644, "chain.pem": 0o644, "fullchain.pem": 0o644,
		"privkey.pem": 0o600, "bundle.pem": 0o600, // bundle includes the key
	}
	for _, a := range arts {
		if a.Mode.Perm() != want[a.Name] {
			t.Errorf("%s mode = %#o, want %#o", a.Name, a.Mode.Perm(), want[a.Name])
		}
	}
}

func TestBuildBundleOmitsAbsentRoot(t *testing.T) {
	m := sampleMaterial()
	m.Root = nil // public CA: no root available
	arts, _ := Build(m, []string{"cert", "chain", "root", "key"})
	want := bytes.Join([][]byte{certPEM("leaf"), certPEM("inter"), keyPEM("key")}, nil)
	if got := find(t, arts, "bundle.pem").Data; !bytes.Equal(got, want) {
		t.Errorf("bundle with absent root\n got: %q\nwant: %q", got, want)
	}
}

func TestBuildBundleOrderAndKeyMode(t *testing.T) {
	arts, _ := Build(sampleMaterial(), []string{"key", "cert", "chain"}) // key first, no root
	b := find(t, arts, "bundle.pem")
	want := bytes.Join([][]byte{keyPEM("key"), certPEM("leaf"), certPEM("inter")}, nil)
	if !bytes.Equal(b.Data, want) {
		t.Errorf("bundle order\n got: %q\nwant: %q", b.Data, want)
	}
	if b.Mode.Perm() != 0o600 {
		t.Errorf("bundle containing key should be 0600, got %#o", b.Mode.Perm())
	}
}

func TestWipe(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"cert.pem", "privkey.pem", "chain.pem", "fullchain.pem", "bundle.pem"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "accounts", "abc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "accounts", "abc", "account.key"), []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := Wipe(dir)
	if err != nil {
		t.Fatalf("Wipe: %v", err)
	}
	if n != 6 { // 5 artifacts + accounts/
		t.Errorf("removed %d, want 6", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "cert.pem")); !os.IsNotExist(err) {
		t.Error("cert.pem should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts")); !os.IsNotExist(err) {
		t.Error("accounts/ should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Error("unrelated keep.txt should remain")
	}
}

func TestWipeEmptyStore(t *testing.T) {
	n, err := Wipe(t.TempDir())
	if err != nil || n != 0 {
		t.Errorf("Wipe(empty) = (%d, %v), want (0, nil)", n, err)
	}
}

func TestWriteAtomicWithModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store") // must be created by Write
	arts := []Artifact{
		{Name: "fullchain.pem", Data: []byte("public"), Mode: 0o644},
		{Name: "privkey.pem", Data: []byte("secret"), Mode: 0o600},
	}
	if err := Write(dir, arts, WriteOptions{}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, a := range arts {
		p := filepath.Join(dir, a.Name)
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", a.Name, err)
		}
		if !bytes.Equal(got, a.Data) {
			t.Errorf("%s content = %q, want %q", a.Name, got, a.Data)
		}
		fi, _ := os.Stat(p)
		if fi.Mode().Perm() != a.Mode.Perm() {
			t.Errorf("%s mode = %#o, want %#o", a.Name, fi.Mode().Perm(), a.Mode.Perm())
		}
	}
	// No leftover temp files.
	entries, _ := os.ReadDir(dir)
	if len(entries) != len(arts) {
		t.Errorf("store dir has %d entries, want %d (stray temp file?)", len(entries), len(arts))
	}
}

func TestWriteDefaultsDirModeTo0700(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	if err := Write(dir, nil, WriteOptions{}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("default store dir mode = %#o, want 0700", fi.Mode().Perm())
	}
}

func TestWriteAppliesDirMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	arts := []Artifact{{Name: "cert.pem", Data: []byte("x"), Mode: 0o644}}
	if err := Write(dir, arts, WriteOptions{DirMode: 0o750}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o750 {
		t.Errorf("store dir mode = %#o, want 0750", fi.Mode().Perm())
	}
}

func TestArchiveCreatesSnapshotPreservingModes(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "cert.pem"), "leaf", 0o644)
	mustWrite(t, filepath.Join(dir, "privkey.pem"), "secret", 0o600)

	if err := Archive(dir, 3); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	snaps := archiveSnaps(t, dir)
	if len(snaps) != 1 {
		t.Fatalf("want 1 snapshot, got %d (%v)", len(snaps), snaps)
	}
	snap := filepath.Join(dir, "archive", snaps[0])
	assertFile(t, filepath.Join(snap, "cert.pem"), "leaf", 0o644)
	assertFile(t, filepath.Join(snap, "privkey.pem"), "secret", 0o600) // key stays locked in the archive
}

func TestArchiveDisabledWhenKeepZero(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "cert.pem"), "leaf", 0o644)
	if err := Archive(dir, 0); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archive")); !os.IsNotExist(err) {
		t.Error("archive dir should not exist when keep=0")
	}
}

func TestArchiveNoopWithoutCurrentCert(t *testing.T) {
	dir := t.TempDir()
	if err := Archive(dir, 3); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archive")); !os.IsNotExist(err) {
		t.Error("no archive should be created when there's no current cert")
	}
}

func TestPruneArchiveKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	for _, ts := range []string{"2024-01-01T00-00-00Z", "2025-01-01T00-00-00Z", "2026-01-01T00-00-00Z"} {
		if err := os.MkdirAll(filepath.Join(dir, "archive", ts), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := pruneArchive(dir, 2); err != nil {
		t.Fatalf("pruneArchive: %v", err)
	}
	got := archiveSnaps(t, dir)
	want := []string{"2025-01-01T00-00-00Z", "2026-01-01T00-00-00Z"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("kept %v, want %v", got, want)
	}
}

func TestWipeCertsKeepsAccounts(t *testing.T) {
	dir := t.TempDir()
	for _, n := range artifactNames {
		mustWrite(t, filepath.Join(dir, n), "x", 0o600)
	}
	mustWrite(t, filepath.Join(dir, "archive", "2026-01-01T00-00-00Z", "cert.pem"), "old", 0o644)
	mustWrite(t, filepath.Join(dir, "accounts", "abc", "account.key"), "k", 0o600)

	n, err := WipeCerts(dir)
	if err != nil {
		t.Fatalf("WipeCerts: %v", err)
	}
	if n != 6 { // 5 artifacts + archive/
		t.Errorf("removed %d, want 6 (5 artifacts + archive)", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "cert.pem")); !os.IsNotExist(err) {
		t.Error("cert.pem should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "archive")); !os.IsNotExist(err) {
		t.Error("archive/ should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts")); err != nil {
		t.Error("accounts/ MUST be kept by WipeCerts")
	}
}

func TestWipeRemovesArchive(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "cert.pem"), "x", 0o644)
	mustWrite(t, filepath.Join(dir, "archive", "2026-01-01T00-00-00Z", "cert.pem"), "old", 0o644)
	if _, err := Wipe(dir); err != nil {
		t.Fatalf("Wipe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archive")); !os.IsNotExist(err) {
		t.Error("archive/ should be removed by Wipe")
	}
}

func mustWrite(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil { // WriteFile respects umask; force the exact mode
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != body {
		t.Errorf("%s content = %q, want %q", path, got, body)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != mode {
		t.Errorf("%s mode = %#o, want %#o", path, fi.Mode().Perm(), mode)
	}
}

func archiveSnaps(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "archive"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
