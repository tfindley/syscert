package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/observe"
	"github.com/tfindley/syscert/internal/renewal"
	"github.com/tfindley/syscert/internal/store"
)

// gatherObservation reads current state off disk into a Snapshot. It is
// read-only and never fails: a missing or unparseable certificate simply yields
// HasCert=false, because "this host has no certificate" is exactly the condition
// monitoring most needs to see.
func gatherObservation(cfg *config.Config, subject string, now time.Time) observe.Snapshot {
	ver, _ := buildInfo()
	s := observe.Snapshot{
		Version:   ver,
		Subject:   subject,
		CA:        cfg.ACME.CA,
		Challenge: cfg.EffectiveChallenge(),
		Generated: now,
		Targets:   make([]observe.Target, 0, len(cfg.Distribute)),
	}

	if certPEM, err := store.ReadCurrentCert(cfg.Store.Path); err == nil {
		if st, leaf, ierr := renewal.Inspect(certPEM, cfg.Renewal.RenewBefore, now); ierr == nil {
			s.HasCert = true
			s.NotBefore, s.NotAfter = st.NotBefore, st.NotAfter
			s.RenewAfter, s.RenewalDue = st.RenewAt, st.Due
			s.Issuer = leaf.Issuer.String()
			s.Serial = leaf.SerialNumber.String()
			s.KeyType = keyType(leaf)
		}
	}

	for _, t := range cfg.Distribute {
		_, err := os.Stat(t.Path)
		s.Targets = append(s.Targets, observe.Target{
			Artifact: t.Artifact,
			Path:     t.Path,
			Present:  err == nil,
		})
	}
	return s
}

// writeObservations writes whichever optional state files are configured. Both
// are off unless a path is set.
//
// A failure here is reported and then swallowed on purpose: these files exist so
// other systems can watch syscert, and an unwritable metrics directory must not
// turn a successful renewal-and-delivery into a failed unit. The warning is
// enough to notice, and dry-run flags the same directories up front.
func writeObservations(cfg *config.Config, subject string, now time.Time, stderr io.Writer) {
	if cfg.Observe.MetricsFile == "" && cfg.Observe.AnsibleFactsFile == "" {
		return
	}
	snap := gatherObservation(cfg, subject, now)

	if p := cfg.Observe.MetricsFile; p != "" {
		if err := observe.Write(p, observe.Prometheus(snap)); err != nil {
			fmt.Fprintf(stderr, "warning: write metrics %s: %v\n", p, err)
		}
	}
	if p := cfg.Observe.AnsibleFactsFile; p != "" {
		data, err := observe.AnsibleFacts(snap)
		if err == nil {
			err = observe.Write(p, data)
		}
		if err != nil {
			fmt.Fprintf(stderr, "warning: write ansible facts %s: %v\n", p, err)
		}
	}
}
