package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/daemon"
)

func main() {
	slog.Warn("agentdserver is deprecated; use agentd --daemon")
	if err := daemon.Run(context.Background()); err != nil {
		slog.Error("Failed to run agentd daemon", "error", err)
		os.Exit(1)
	}
}
