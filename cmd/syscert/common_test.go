package main

import (
	"flag"
	"testing"
)

// resolveConfig registers configFlag on a fresh flag set, parses args, and
// returns the resolved config path — mirroring how each command resolves it.
func resolveConfig(t *testing.T, args []string) string {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	p := configFlag(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return *p
}

func TestConfigFlagPrecedence(t *testing.T) {
	t.Run("default when env unset and no flag", func(t *testing.T) {
		t.Setenv(envConfig, "")
		if got := resolveConfig(t, nil); got != defaultConfigPath {
			t.Errorf("got %q, want default %q", got, defaultConfigPath)
		}
	})

	t.Run("env used when set and no flag", func(t *testing.T) {
		t.Setenv(envConfig, "/tmp/from-env.toml")
		if got := resolveConfig(t, nil); got != "/tmp/from-env.toml" {
			t.Errorf("got %q, want env value", got)
		}
	})

	t.Run("flag overrides env", func(t *testing.T) {
		t.Setenv(envConfig, "/tmp/from-env.toml")
		if got := resolveConfig(t, []string{"--config", "/tmp/from-flag.toml"}); got != "/tmp/from-flag.toml" {
			t.Errorf("got %q, want flag value", got)
		}
	})
}
