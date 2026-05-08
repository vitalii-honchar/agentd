package config

import (
	"log/slog"
	"os"
)

func ConfigureLogger(cfg *Config) {
	level := slog.LevelDebug
	if cfg != nil && cfg.Production {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}
