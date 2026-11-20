// Package logging builds the one logger both binaries use.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a slog logger. A level or format this build does not know falls back
// rather than failing the boot: logging is never the reason a server refuses to start.
func New(level, format string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: l}
	var handler slog.Handler
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}
