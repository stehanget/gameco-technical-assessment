package logger

import (
	"log/slog"
	"os"
)

func New(name, env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	if env == "test" {
		level = slog.LevelWarn
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With(
		"service", name,
		"environment", env,
	)
}
