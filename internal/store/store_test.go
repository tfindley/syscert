package store

import (
	"bytes"
	"encoding/pem"
	"os"
	"path/filepath"
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
	if err := Write(dir, arts); err != nil {
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
