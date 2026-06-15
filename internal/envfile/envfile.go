// Package envfile loads environment variables from a systemd EnvironmentFile-style
// file, so a manual `syscert` run can pick up the same /etc/syscert/secrets the
// systemd unit reads via EnvironmentFile= — without exporting every variable by
// hand. It supports KEY=value lines, # and ; comments, blank lines, and optional
// surrounding single/double quotes. Values are never logged.
package envfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// KV is a parsed key/value pair, in file order.
type KV struct {
	Key   string
	Value string
}

// keyRE matches a valid environment-variable name.
var keyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Parse reads KEY=value pairs (systemd EnvironmentFile syntax) from r. It returns
// the pairs in file order, or an error citing the offending line number. Error
// messages never include a line's value, only its number.
func Parse(r io.Reader) ([]KV, error) {
	var out []KV
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		raw := strings.TrimLeft(sc.Text(), " \t")
		if raw == "" || raw[0] == '#' || raw[0] == ';' {
			continue
		}
		eq := strings.IndexByte(raw, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' (expected KEY=value)", line)
		}
		key := strings.TrimSpace(raw[:eq])
		if !keyRE.MatchString(key) {
			return nil, fmt.Errorf("line %d: invalid variable name", line)
		}
		out = append(out, KV{Key: key, Value: unquote(strings.TrimSpace(raw[eq+1:]))})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// unquote strips one layer of matching surrounding single or double quotes,
// preserving whatever is inside them verbatim.
func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// merge resolves the variables to set given the keys already present in the
// environment (preset) and the parsed files in order. Variables already in the
// environment are skipped (real env wins); among files and duplicate lines, the
// last value wins; the result is ordered by each key's first appearance.
func merge(preset map[string]bool, files [][]KV) []KV {
	idx := map[string]int{}
	var out []KV
	for _, kvs := range files {
		for _, kv := range kvs {
			if preset[kv.Key] {
				continue
			}
			if i, ok := idx[kv.Key]; ok {
				out[i].Value = kv.Value
				continue
			}
			idx[kv.Key] = len(out)
			out = append(out, kv)
		}
	}
	return out
}

// Load parses each path and applies the variables to the process environment,
// without overwriting variables already set (so an explicit export still wins).
// It returns the names of the variables it set, never their values.
func Load(paths []string) ([]string, error) {
	files := make([][]KV, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p) // #nosec G304 -- operator-provided --env-file path (trusted CLI input)
		if err != nil {
			return nil, err
		}
		kvs, perr := Parse(bytes.NewReader(data))
		if perr != nil {
			return nil, fmt.Errorf("%s: %w", p, perr)
		}
		files = append(files, kvs)
	}

	preset := map[string]bool{}
	for _, e := range os.Environ() {
		if i := strings.IndexByte(e, '='); i >= 0 {
			preset[e[:i]] = true
		}
	}

	resolved := merge(preset, files)
	set := make([]string, 0, len(resolved))
	for _, kv := range resolved {
		if err := os.Setenv(kv.Key, kv.Value); err != nil {
			return set, err
		}
		set = append(set, kv.Key)
	}
	return set, nil
}
