package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"":      slog.LevelInfo, // default
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"WARN":  slog.LevelWarn, // case-insensitive
		"bogus": slog.LevelInfo, // unknown → info
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewTextFormat(t *testing.T) {
	var buf bytes.Buffer
	New("info", "text", &buf).Info("hello", "subject", "host.example.com")
	out := buf.String()
	if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "hello") || !strings.Contains(out, "subject=host.example.com") {
		t.Errorf("text output missing expected fields: %q", out)
	}
}

func TestNewJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	New("info", "json", &buf).Info("hello")
	out := buf.String()
	if !strings.Contains(out, `"level":"INFO"`) || !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("json output missing expected fields: %q", out)
	}
}

func TestNewLevelSuppresses(t *testing.T) {
	var buf bytes.Buffer
	l := New("warn", "text", &buf)
	l.Info("should be suppressed")
	l.Warn("should appear")
	out := buf.String()
	if strings.Contains(out, "suppressed") {
		t.Errorf("info record should be suppressed at warn level: %q", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("warn record should appear: %q", out)
	}
}
