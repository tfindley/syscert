// Package renewal decides whether a stored certificate is due for renewal.
package renewal

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Status describes a stored leaf certificate's renewal state.
type Status struct {
	NotBefore time.Time // issued
	NotAfter  time.Time // expires
	RenewAt   time.Time // renewal becomes due at/after this (NotAfter − window)
	Due       bool      // now is within the renewal window (or past expiry)
}

// Inspect parses the leaf certificate in certPEM and computes its renewal Status
// at now, also returning the parsed leaf so callers needn't re-parse it. The
// renewal window is renewBefore when set, otherwise one third of the certificate's
// total lifetime (ADR-0022).
func Inspect(certPEM []byte, renewBefore string, now time.Time) (Status, *x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return Status{}, nil, fmt.Errorf("no PEM certificate block found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Status{}, nil, fmt.Errorf("parse certificate: %w", err)
	}

	window := cert.NotAfter.Sub(cert.NotBefore) / 3 // auto: ⅓ of lifetime
	if renewBefore != "" {
		window, err = parseWindow(renewBefore)
		if err != nil {
			return Status{}, nil, err
		}
	}
	renewAt := cert.NotAfter.Add(-window)
	return Status{
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		RenewAt:   renewAt,
		Due:       now.After(renewAt),
	}, cert, nil
}

// Due reports whether the leaf certificate in certPEM should be renewed now. A
// cert is due once now is within the renewal window of its NotAfter (or past it).
func Due(certPEM []byte, renewBefore string, now time.Time) (bool, error) {
	s, _, err := Inspect(certPEM, renewBefore, now)
	if err != nil {
		return false, err
	}
	return s.Due, nil
}

// parseWindow parses a renewal window: a Go duration (e.g. "48h", "90m") or a
// whole number of days with a "d" suffix (e.g. "30d").
func parseWindow(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration %q: %w", s, err)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return d, nil
}
