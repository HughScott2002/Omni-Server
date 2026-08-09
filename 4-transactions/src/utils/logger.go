package utils

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger installs the process-wide slog logger and should be the first
// thing main does.
//
// LOG_LEVEL (debug, info, warn, error) sets the threshold and defaults to
// info, so a service can be turned up or quietened on a running container
// without a rebuild. Local runs get the text handler because it stays
// readable in `docker compose logs`; everything else gets JSON.
//
// This is duplicated per service because each one is its own Go module with
// no shared library yet. It should collapse into libs/ with the Kafka and
// Redis setup (#12).
func InitLogger(service string) {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.ToLower(os.Getenv("ENVIRONMENT")) == "local" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler).With("service", service))
}
