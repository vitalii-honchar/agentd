package main

import (
	"context"
	"fmt"
	"os"

	cliapp "github.com/vitalii-honchar/agentd/internal/agentd/app"
	"github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/internal/agentd/infra/httpclient"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client, err := httpclient.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := cliapp.NewRootCommand(cliapp.RootOptions{
		Config:        cfg,
		Client:        client,
		ExecuteClient: client,
		StopClient:    client,
		QueryClient:   client,
	})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cliapp.ExitCode(err))
	}
}
