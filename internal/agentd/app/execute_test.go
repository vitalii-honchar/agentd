package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"

	"github.com/spf13/cobra"
)

func TestExecuteCommandCallsClient(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "release-notes-helper", Status: "running",
	}}
	var out bytes.Buffer
	cmd := NewExecuteCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.executeAgent != "release-notes-helper" {
		t.Fatalf("execute agent: got %q", client.executeAgent)
	}
	if !strings.Contains(out.String(), "running release-notes-helper run-1") {
		t.Fatalf("output: got %q", out.String())
	}
}

func TestStopCommandCallsClientWithRunID(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "release-notes-helper", Status: "stopping",
	}}
	var out bytes.Buffer
	cmd := NewStopCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper", "--run", "run-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.stopRequest.AgentName != "release-notes-helper" || client.stopRequest.RunID != "run-1" {
		t.Fatalf("stop request: %#v", client.stopRequest)
	}
}

func TestRootCommandWiresRuntimeCommands(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{}
	cmd := NewRootCommand(RootOptions{
		Config: &config.Config{
			ServerURL:      config.DefaultServerURL,
			OutputFormat:   config.OutputText,
			RequestTimeout: config.DefaultRequestTimeout,
		},
		ExecuteClient: client,
		StopClient:    client,
		Out:           &bytes.Buffer{},
		Err:           &bytes.Buffer{},
	})

	requireCommand(t, cmd, "execute")
	requireCommand(t, cmd, "stop")
}

type fakeRuntimeClient struct {
	executeAgent string
	stopRequest  StopRequest
	run          RunResponse
	err          error
}

func (f *fakeRuntimeClient) Execute(_ context.Context, agentName string) (RunResponse, error) {
	f.executeAgent = agentName
	if f.err != nil {
		return RunResponse{}, f.err
	}

	return f.run, nil
}

func (f *fakeRuntimeClient) Stop(_ context.Context, request StopRequest) (RunResponse, error) {
	f.stopRequest = request
	if f.err != nil {
		return RunResponse{}, f.err
	}

	return f.run, nil
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
