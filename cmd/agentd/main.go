package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cliapp "github.com/vitalii-honchar/agentd/internal/agentd/app"
	"github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/internal/agentd/infra/httpclient"
)

type agentdMode string

const (
	agentdModeClient agentdMode = "client"
	agentdModeDaemon agentdMode = "daemon"
)

var (
	errDaemonWithSubcommand = errors.New("daemon mode cannot be combined with a client subcommand")
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cliapp.ExitCode(err))
	}
}

func run(ctx context.Context, args []string) error {
	mode, err := agentdModeFromArgs(args)
	if err != nil {
		return err
	}
	if mode == agentdModeDaemon {
		return runDaemon(ctx)
	}

	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	client, err := httpclient.New(cfg)
	if err != nil {
		return err
	}

	cmd := cliapp.NewRootCommand(cliapp.RootOptions{
		Config:        cfg,
		Client:        client,
		ExecuteClient: client,
		StopClient:    client,
		QueryClient:   client,
	})
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(ctx); err != nil {
		return err
	}

	return nil
}

func agentdModeFromArgs(args []string) (agentdMode, error) {
	daemon := false
	for _, arg := range args {
		switch arg {
		case "--daemon", "-d", "--deamon":
			daemon = true
		default:
			if daemon {
				return "", errDaemonWithSubcommand
			}
		}
	}
	if daemon {
		return agentdModeDaemon, nil
	}

	return agentdModeClient, nil
}
