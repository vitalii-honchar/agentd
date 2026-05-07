package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	agentdserver "agentd/internal/agentdserver"
)

func main() {
	server, err := agentdserver.New()
	if err != nil {
		slog.Error("Failed to create agentdserver", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Start(ctx); err != nil {
		slog.Error("Failed to start agentdserver", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), server.Config.Runtime.ShutdownTimeout)
	defer cancel()
	if err := server.Stop(shutdownCtx); err != nil {
		slog.Error("Failed to stop agentdserver", "error", err)
		os.Exit(1)
	}
}
