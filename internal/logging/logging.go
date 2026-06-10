// Package logging builds the structured logger for SysCert's operational output
// (events + errors, on stderr), separate from a command's human-facing results
// on stdout. Format and level come from the [logging] config (ADR-0012).
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New returns an slog.Logger writing to w at the given level, in "text" (default,
// journald-friendly) or "json" format.
func New(level, format string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	var h slog.Handler
	if strings.EqualFold(format, "json") {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}

// parseLevel maps a config level string to an slog.Level (default info).
func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
