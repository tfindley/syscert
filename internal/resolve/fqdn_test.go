package resolve

import (
	"errors"
	"testing"
)

func constHostname(name string, err error) func() (string, error) {
	return func() (string, error) { return name, err }
}

func TestResolveUsesOverride(t *testing.T) {
	// An explicit override wins, regardless of the system hostname.
	got, err := FQDN("host.example.com", constHostname("ignored-short", nil))
	if err != nil {
		t.Fatalf("FQDN: %v", err)
	}
	if got != "host.example.com" {
		t.Errorf("FQDN = %q, want %q", got, "host.example.com")
	}
}

func TestResolveUsesSystemFQDN(t *testing.T) {
	// With no override, a system name that is a FQDN (has a domain) is used.
	got, err := FQDN("", constHostname("box.internal.lan", nil))
	if err != nil {
		t.Fatalf("FQDN: %v", err)
	}
	if got != "box.internal.lan" {
		t.Errorf("FQDN = %q, want %q", got, "box.internal.lan")
	}
}

func TestResolveErrorsWhenNoFQDN(t *testing.T) {
	// A bare short hostname is not a FQDN — ADR-0004 says error, never guess.
	if _, err := FQDN("", constHostname("box", nil)); err == nil {
		t.Fatal("FQDN with short hostname: want error, got nil")
	}
}

func TestResolveErrorsWhenHostnameUnavailable(t *testing.T) {
	if _, err := FQDN("", constHostname("", errors.New("boom"))); err == nil {
		t.Fatal("FQDN with hostname lookup failure: want error, got nil")
	}
}
