// Package observe renders syscert's current state into files other tools read:
// a Prometheus node_exporter textfile and an Ansible local-facts file. Both are
// opt-in (see config.ObserveConfig), write-only, and never read back — nothing in
// syscert changes behaviour because of them.
//
// Rendering is separated from gathering on purpose: everything here is a pure
// function of a Snapshot, so the exact bytes are unit-tested without a store, a
// certificate, or a filesystem.
package observe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tfindley/syscert/internal/atomicfile"
)

// Target is one distribution target's observed state.
type Target struct {
	Artifact string `json:"artifact"`
	Path     string `json:"path"`
	Present  bool   `json:"present"`
}

// Snapshot is everything the outputs can report. Zero values are fine: a host
// with no certificate yet still produces a valid file saying exactly that, which
// is more useful to an alert than a missing file.
type Snapshot struct {
	Version   string    `json:"version"`
	Subject   string    `json:"subject"`
	CA        string    `json:"ca"`
	Challenge string    `json:"challenge"`
	HasCert   bool      `json:"has_cert"`
	KeyType   string    `json:"key_type,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	Serial    string    `json:"serial,omitempty"`
	NotBefore time.Time `json:"not_before,omitempty"`
	NotAfter  time.Time `json:"not_after,omitempty"`
	// RenewAfter is when the certificate becomes eligible for renewal.
	RenewAfter time.Time `json:"renew_after,omitempty"`
	RenewalDue bool      `json:"renewal_due"`
	Targets    []Target  `json:"targets"`
	Generated  time.Time `json:"generated"`
}

// Prometheus renders the node_exporter textfile-collector exposition.
//
// Timestamps are emitted as plain unix-seconds gauges rather than as Prometheus
// sample timestamps: a sample timestamp would make the series look stale to the
// scraper, whereas a gauge of "when does this expire" is what an alert actually
// wants — typically (syscert_cert_not_after_seconds - time()) / 86400.
func Prometheus(s Snapshot) []byte {
	var b strings.Builder
	g := func(name, help string, v int64) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, v)
	}

	g("syscert_last_run_timestamp_seconds", "Unix time of the last completed syscert run.", s.Generated.Unix())
	g("syscert_cert_present", "1 if a certificate is present in the store, 0 otherwise.", b2i(s.HasCert))

	if s.HasCert {
		g("syscert_cert_not_before_seconds", "Certificate notBefore, unix seconds.", s.NotBefore.Unix())
		g("syscert_cert_not_after_seconds", "Certificate notAfter (expiry), unix seconds.", s.NotAfter.Unix())
		if !s.RenewAfter.IsZero() {
			g("syscert_cert_renew_after_seconds", "When the certificate becomes eligible for renewal, unix seconds.", s.RenewAfter.Unix())
		}
		g("syscert_cert_renewal_due", "1 if the certificate is inside its renewal window.", b2i(s.RenewalDue))
	}

	var present int64
	for _, t := range s.Targets {
		if t.Present {
			present++
		}
	}
	g("syscert_distribute_targets", "Configured distribution targets.", int64(len(s.Targets)))
	g("syscert_distribute_targets_present", "Distribution targets whose file exists on disk.", present)

	// Per-target presence, so an alert can name the consumer that is missing one.
	if len(s.Targets) > 0 {
		b.WriteString("# HELP syscert_distribute_target_present 1 if this target's file exists on disk.\n")
		b.WriteString("# TYPE syscert_distribute_target_present gauge\n")
		for _, t := range s.Targets {
			fmt.Fprintf(&b, "syscert_distribute_target_present{path=\"%s\",artifact=\"%s\"} %d\n",
				escapeLabel(t.Path), escapeLabel(t.Artifact), b2i(t.Present))
		}
	}

	// An info metric carries the strings; the value is always 1 by convention.
	b.WriteString("# HELP syscert_cert_info Labelled certificate metadata; value is always 1.\n")
	b.WriteString("# TYPE syscert_cert_info gauge\n")
	fmt.Fprintf(&b, "syscert_cert_info{subject=\"%s\",ca=\"%s\",challenge=\"%s\",issuer=\"%s\","+
		"serial=\"%s\",key_type=\"%s\",version=\"%s\"} 1\n",
		escapeLabel(s.Subject), escapeLabel(s.CA), escapeLabel(s.Challenge),
		escapeLabel(s.Issuer), escapeLabel(s.Serial), escapeLabel(s.KeyType), escapeLabel(s.Version))

	return []byte(b.String())
}

// AnsibleFacts renders the local-facts JSON. A non-executable file under
// /etc/ansible/facts.d is parsed by Ansible and surfaces as ansible_local.syscert.
func AnsibleFacts(s Snapshot) ([]byte, error) {
	if s.Targets == nil {
		s.Targets = []Target{} // render [] rather than null, which is friendlier to filters
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Write places data at path atomically and world-readable. Atomic matters: both
// consumers poll these files on their own schedule, and node_exporter explicitly
// requires the rename-into-place pattern so it never reads a half-written file.
// 0644 because the readers (node_exporter, Ansible) are not the syscert user.
func Write(path string, data []byte) error {
	return atomicfile.Write(path, data, 0o644, nil)
}

func b2i(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// escapeLabel escapes a Prometheus label value: backslash, double quote and
// newline, per the exposition format. A path or issuer CN can legitimately
// contain a backslash or quote, and an unescaped one corrupts the whole file.
func escapeLabel(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}
