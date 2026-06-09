// Package resolve determines the certificate subject (system FQDN or override).
package resolve

import (
	"fmt"
	"os"
	"strings"
)

// HostnameFunc returns the system hostname. Injectable for testing.
type HostnameFunc func() (string, error)

// FQDN returns the certificate subject name.
//
// If override is non-empty it is used as-is. Otherwise the system hostname is
// consulted via hostnameFn; it is accepted only if it is a fully-qualified name
// (contains a domain). If no FQDN can be determined, FQDN returns an error
// rather than guessing (ADR-0004).
//
// A nil hostnameFn defaults to os.Hostname.
func FQDN(override string, hostnameFn HostnameFunc) (string, error) {
	if override != "" {
		return override, nil
	}
	if hostnameFn == nil {
		hostnameFn = os.Hostname
	}
	name, err := hostnameFn()
	if err != nil {
		return "", fmt.Errorf("determine system hostname: %w", err)
	}
	name = strings.TrimSpace(name)
	if !strings.Contains(strings.TrimSuffix(name, "."), ".") {
		return "", fmt.Errorf("system hostname %q is not a FQDN; set cert.hostname in the config", name)
	}
	return name, nil
}
