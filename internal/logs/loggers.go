package logs

import (
	"io"
	"log/slog"
	"os"
)

// Config controls logger initialization.
type Config struct {
	Level string
	JSON  bool
	Out   io.Writer
}

// New creates a configured slog logger.
func New(cfg Config) *slog.Logger {
	out := cfg.Out
	if out == nil {
		out = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	if cfg.JSON {
		return slog.New(slog.NewJSONHandler(out, opts))
	}
	return slog.New(slog.NewTextHandler(out, opts))
}

// Default returns the package default logger.
func Default() *slog.Logger {
	return New(Config{Level: "info"})
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}