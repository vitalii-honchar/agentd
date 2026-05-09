package main

import (
	"errors"
	"testing"
)

func TestAgentdModeFromArgsDetectsDaemonFlags(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"long daemon":       {"--daemon"},
		"short daemon":      {"-d"},
		"compat misspelled": {"--deamon"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mode, err := agentdModeFromArgs(args)
			if err != nil {
				t.Fatalf("agentdModeFromArgs: %v", err)
			}
			if mode != agentdModeDaemon {
				t.Fatalf("mode: got %q want %q", mode, agentdModeDaemon)
			}
		})
	}
}

func TestAgentdModeFromArgsDefaultsToClient(t *testing.T) {
	t.Parallel()

	mode, err := agentdModeFromArgs([]string{"list"})
	if err != nil {
		t.Fatalf("agentdModeFromArgs: %v", err)
	}
	if mode != agentdModeClient {
		t.Fatalf("mode: got %q want %q", mode, agentdModeClient)
	}
}

func TestAgentdModeFromArgsRejectsDaemonWithSubcommand(t *testing.T) {
	t.Parallel()

	_, err := agentdModeFromArgs([]string{"--daemon", "list"})
	if !errors.Is(err, errDaemonWithSubcommand) {
		t.Fatalf("agentdModeFromArgs error: got %v want %v", err, errDaemonWithSubcommand)
	}
}
