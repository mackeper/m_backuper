package logger

import (
	"log/slog"
	"os"
)

// New creates a new structured logger with the specified verbosity level
func New(verbose bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if verbose {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// NewJSON creates a JSON-formatted logger (useful for machine-readable logs)
func NewJSON(verbose bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if verbose {
		opts.Level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
