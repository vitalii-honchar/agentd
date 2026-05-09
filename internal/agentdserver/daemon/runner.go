package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	agentdserver "github.com/vitalii-honchar/agentd/internal/agentdserver"
)

func Run(ctx context.Context) error {
	server, err := agentdserver.New()
	if err != nil {
		return fmt.Errorf("create agentdserver: %w", err)
	}

	runCtx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := server.Start(runCtx); err != nil {
		return fmt.Errorf("start agentdserver: %w", err)
	}

	<-runCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), server.Config.Runtime.ShutdownTimeout)
	defer cancel()
	if err := server.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("stop agentdserver: %w", err)
	}

	return nil
}
