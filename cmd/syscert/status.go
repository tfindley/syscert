package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/renewal"
	"github.com/tfindley/syscert/internal/resolve"
	"github.com/tfindley/syscert/internal/store"
	"github.com/tfindley/syscert/internal/validate"
)

// cmdStatus prints a human-readable, OFFLINE snapshot of SysCert's current state:
// resolved config, the stored certificate's dates, the ACME account(s), archived
// snapshots, and distribution targets. No network, no credentials, no secrets.
func cmdStatus(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("status", stderr)
	cfgPath := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "status: load config: %v\n", err)
		return 2
	}
	writeStatus(stdout, cfg, time.Now())
	return 0
}

// writeStatus renders the report for cfg at now (split out for testing). It is
// lenient — it shows whatever state exists even when the config has problems.
func writeStatus(w io.Writer, cfg *config.Config, now time.Time) {
	subject, ferr := resolve.FQDN(cfg.Cert.Hostname, nil)
	if ferr != nil {
		subject = "(unresolved: " + ferr.Error() + ")"
	}
	fmt.Fprintf(w, "subject:   %s\n", subject)
	fmt.Fprintf(w, "CA:        %s\n", cfg.ACME.CA)
	if config.IsInternalCA(cfg.ACME.CA) && cfg.ACME.DirectoryURL != "" {
		fmt.Fprintf(w, "directory: %s\n", cfg.ACME.DirectoryURL)
	}
	fmt.Fprintf(w, "challenge: %s\n", cfg.EffectiveChallenge())
	eab := "no"
	if cfg.ACME.EAB.Kid != "" {
		eab = "yes (kid set)" // never print the HMAC — it's a secret from the env
	}
	fmt.Fprintf(w, "EAB:       %s\n", eab)
	if ps := validate.Config(cfg); len(ps) > 0 {
		fmt.Fprintf(w, "config:    %d problem(s) — run `syscert dry-run --config-only`\n", len(ps))
	}

	fmt.Fprintln(w, "\ncertificate:")
	if certPEM, err := store.ReadCurrentCert(cfg.Store.Path); err != nil {
		fmt.Fprintf(w, "  none yet in %s\n", cfg.Store.Path)
	} else if st, leaf, ierr := renewal.Inspect(certPEM, cfg.Renewal.RenewBefore, now); ierr != nil {
		fmt.Fprintf(w, "  unreadable: %v\n", ierr)
	} else {
		fmt.Fprintf(w, "  subject:    %s\n", leaf.Subject.CommonName)
		fmt.Fprintf(w, "  SANs:       %s\n", sans(leaf))
		fmt.Fprintf(w, "  issuer:     %s\n", leaf.Issuer.CommonName)
		fmt.Fprintf(w, "  key:        %s\n", keyType(leaf))
		fmt.Fprintf(w, "  issued:     %s\n", st.NotBefore.Format(time.RFC3339))
		fmt.Fprintf(w, "  expires:    %s (in %s)\n", st.NotAfter.Format(time.RFC3339), humanDur(st.NotAfter.Sub(now)))
		fmt.Fprintf(w, "  renews:     %s\n", renewsPhrase(st, now))
	}

	fmt.Fprintf(w, "\naccounts:  %d (under %s/accounts)\n", store.CountAccounts(cfg.Store.Path), cfg.Store.Path)
	if snaps := store.ListArchive(cfg.Store.Path); len(snaps) > 0 {
		fmt.Fprintf(w, "archive:   %d snapshot(s)\n", len(snaps))
	}

	if len(cfg.Distribute) > 0 {
		fmt.Fprintln(w, "\ndistribute:")
		for _, d := range cfg.Distribute {
			state := "present"
			if _, err := os.Stat(d.Path); err != nil {
				state = "MISSING"
			}
			fmt.Fprintf(w, "  %-10s %s  [%s]\n", d.Artifact, d.Path, state)
		}
	}
}

// certLine is the one-line cert summary `ensure` logs on completion, so it surfaces
// in `systemctl status syscert` / `journalctl -u syscert`. Empty when no readable cert.
func certLine(cfg *config.Config, now time.Time) string {
	certPEM, err := store.ReadCurrentCert(cfg.Store.Path)
	if err != nil {
		return ""
	}
	st, _, err := renewal.Inspect(certPEM, cfg.Renewal.RenewBefore, now)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("cert expires %s (in %s); renews %s",
		st.NotAfter.Format("2006-01-02"), humanDur(st.NotAfter.Sub(now)), renewsPhrase(st, now))
}

// renewsPhrase renders a Status's renewal timing as "due now" or "in <duration>".
func renewsPhrase(st renewal.Status, now time.Time) string {
	if st.Due {
		return "due now"
	}
	return "in " + humanDur(st.RenewAt.Sub(now))
}

func sans(c *x509.Certificate) string {
	out := append([]string{}, c.DNSNames...)
	for _, ip := range c.IPAddresses {
		out = append(out, ip.String())
	}
	if len(out) == 0 {
		return "(none)"
	}
	return strings.Join(out, ", ")
}

func keyType(c *x509.Certificate) string {
	switch k := c.PublicKey.(type) {
	case *ecdsa.PublicKey:
		return "ECDSA " + k.Curve.Params().Name
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d", k.N.BitLen())
	default:
		return c.PublicKeyAlgorithm.String()
	}
}

// humanDur renders a non-negative duration as "Nd", "Nd Mh", or "Mh"; negatives
// render as "expired".
func humanDur(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	days, hours := int(d.Hours())/24, int(d.Hours())%24
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	default:
		return fmt.Sprintf("%dh", hours)
	}
}
