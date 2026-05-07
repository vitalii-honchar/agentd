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
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return attr
		},
	})

	slog.SetDefault(slog.New(handler))
}
