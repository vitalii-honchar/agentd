package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func testCLIConfig() *config.Config {
	return &config.Config{
		ServerURL:      config.DefaultServerURL,
		OutputFormat:   config.OutputText,
		RequestTimeout: config.DefaultRequestTimeout,
	}
}

func executeTestCommand(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()

	cmd.SetArgs(args)
	return cmd.Execute()
}

func requireCommand(t *testing.T, cmd interface{ Commands() []*cobra.Command }, name string) {
	t.Helper()
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return
		}
	}
	t.Fatalf("command %q was not wired", name)
}

func requireOutputContains(t *testing.T, out *bytes.Buffer, expected string) {
	t.Helper()
	if !strings.Contains(out.String(), expected) {
		t.Fatalf("output missing %q: %q", expected, out.String())
	}
}
