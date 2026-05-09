package main

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/daemon"
)

func runDaemon(ctx context.Context) error {
	return daemon.Run(ctx)
}
