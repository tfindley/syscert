package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	legolog "github.com/go-acme/lego/v5/log"
	"github.com/tfindley/syscert/internal/acme"
	"github.com/tfindley/syscert/internal/config"
	"github.com/tfindley/syscert/internal/logging"
	"github.com/tfindley/syscert/internal/resolve"
	"github.com/tfindley/syscert/internal/store"
	"github.com/tfindley/syscert/internal/validate"
)

// defaultConfigPath is used when --config is not given.
const defaultConfigPath = "/etc/syscert/syscert.toml"

// loadConfig loads the config and installs the structured logger from it, so
// every command logs consistently the moment its config is available — whether
// or not it goes on to run full validation. This is the single seam where
// logging is wired up.
func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	l := logging.New(cfg.Logging.Level, cfg.Logging.Format, os.Stderr)
	slog.SetDefault(l)    // operational events + errors → stderr, honouring [logging]
	legolog.SetDefault(l) // route lego's ACME logs through the same logger (ADR-0012)
	return cfg, nil
}

// loadAndCheck loads the config, resolves the certificate subject, and runs the
// fail-fast validation. A config-load failure is returned as err; everything
// else surfaces as problems (empty == ready to proceed).
func loadAndCheck(cfgPath string) (cfg *config.Config, subject string, problems []validate.Problem, err error) {
	cfg, err = loadConfig(cfgPath)
	if err != nil {
		return nil, "", nil, err
	}
	subject, ferr := resolve.FQDN(cfg.Cert.Hostname, nil)
	if ferr != nil {
		problems = append(problems, validate.Problem{Field: "cert.hostname", Message: ferr.Error()})
	}
	problems = append(problems, validate.Config(cfg)...)
	return cfg, subject, problems, nil
}

// printProblems writes a FAIL summary with one line per problem.
func printProblems(w io.Writer, problems []validate.Problem) {
	fmt.Fprintf(w, "FAIL: %d config problem(s)\n", len(problems))
	for _, p := range problems {
		fmt.Fprintf(w, "  - %s: %s\n", p.Field, p.Message)
	}
}

// confirm prints prompt to w and reads a yes/no answer from in (default No).
// force short-circuits to true without prompting — used by --force.
func confirm(w io.Writer, in io.Reader, prompt string, force bool) bool {
	if force {
		return true
	}
	fmt.Fprint(w, prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// noteConnectionTrust warns when the ACME connection trusts an internal CA via
// ca_bundle rather than the system trust store — a deliberate, connection-only
// trust decision the operator should be aware of (ADR-0035).
func noteConnectionTrust(cfg *config.Config, w io.Writer) {
	if cfg.ACME.CABundle != "" {
		fmt.Fprintf(w, "warning: trusting the ACME server via ca_bundle %s for this connection only "+
			"(not the system trust store) — run 'syscert trust install' to trust issued certs host-wide\n",
			cfg.ACME.CABundle)
	}
}

// provision runs the obtain + persist stages of the pipeline: it obtains a
// certificate and writes the five artifacts to the store. It does NOT
// distribute (callers decide). Shared by issue, renew, and ensure.
func provision(cfg *config.Config, subject string, staging bool) ([]store.Artifact, error) {
	res, err := acme.Obtain(context.Background(), cfg, subject, acme.NewLegoObtainer(), staging)
	if err != nil {
		return nil, err
	}
	arts, err := store.Build(store.Material{
		Certificate:       res.Certificate,
		PrivateKey:        res.PrivateKey,
		IssuerCertificate: res.IssuerCertificate,
		// Root is only available from internal CAs; sourcing it is a later increment.
	}, cfg.Bundle.Order)
	if err != nil {
		return nil, err
	}
	if err := store.Write(cfg.Store.Path, arts); err != nil {
		return nil, err
	}
	slog.Info("certificate provisioned",
		"subject", subject, "ca", cfg.ACME.CA, "challenge", cfg.EffectiveChallenge())
	return arts, nil
}
