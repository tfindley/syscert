package main

import "testing"

// A drop-in is a systemd unit fragment: whitespace separates paths and a newline
// injects a directive. The installers refuse these shapes; so must the binary
// that writes the file, or a typo silently widens ProtectSystem=strict.
func TestUnsafeGrantDir(t *testing.T) {
	safe := []string{"/etc/nginx/tls", "/etc/cockpit/ws-certs.d", "/var/lib/node_exporter/textfile_collector", "/opt/a/b/c"}
	for _, d := range safe {
		if why := unsafeGrantDir(d); why != "" {
			t.Errorf("unsafeGrantDir(%q) = %q, want it allowed", d, why)
		}
	}
	unsafe := map[string]string{
		"/etc":                            "top-level directory",
		"/usr":                            "top-level directory",
		"/":                               "root",
		"relative/dir":                    "not absolute",
		"/etc/my certs":                   "whitespace splits the directive",
		"/etc/a\nExecStartPre=/bin/false": "newline injects a directive",
		"/etc/a\tb":                       "tab is whitespace",
	}
	for d, reason := range unsafe {
		if unsafeGrantDir(d) == "" {
			t.Errorf("unsafeGrantDir(%q) allowed it; must be refused (%s)", d, reason)
		}
	}
}

// The store is syscert's own directory and is always kept, even though callers
// may pass it alongside operator-supplied paths.
func TestPartitionGrantDirsKeepsStoreAndRefusesBroad(t *testing.T) {
	store := "/var/lib/syscert"
	ok, refused := partitionGrantDirs([]string{store, "/etc/nginx/tls", "/etc", "bad/rel"}, store)

	want := map[string]bool{store: true, "/etc/nginx/tls": true}
	if len(ok) != len(want) {
		t.Fatalf("granted %v, want %v", ok, want)
	}
	for _, d := range ok {
		if !want[d] {
			t.Errorf("unexpectedly granted %q", d)
		}
	}
	for _, d := range []string{"/etc", "bad/rel"} {
		if _, found := refused[d]; !found {
			t.Errorf("%q should have been refused", d)
		}
	}
}
