package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
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

func TestExecuteCommandPassesInputs(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "website-snapshot-analyst", Status: "running",
	}}
	var out bytes.Buffer
	cmd := NewExecuteCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"website-snapshot-analyst", "--input", "url=https://example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.executeInputs["url"] != "https://example.com" {
		t.Fatalf("execute inputs: %#v", client.executeInputs)
	}
}

func TestExecuteCommandPassesInputJSON(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "contracted-agent", Status: "running",
	}}
	var out bytes.Buffer
	cmd := NewExecuteCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"contracted-agent", "--input-json", `{"topic":"agentd","limit":3}`})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var input map[string]any
	if err := json.Unmarshal(client.executeRunInput.Input, &input); err != nil {
		t.Fatalf("decode input JSON: %v raw=%s", err, client.executeRunInput.Input)
	}
	if input["topic"] != "agentd" || input["limit"] != float64(3) {
		t.Fatalf("input JSON: %#v", input)
	}
}

func TestExecuteCommandPassesInputFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"topic":"agentd"}`), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}
	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "contracted-agent", Status: "running",
	}}
	var out bytes.Buffer
	cmd := NewExecuteCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"contracted-agent", "--input-file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if string(client.executeRunInput.Input) != `{"topic":"agentd"}` {
		t.Fatalf("input file JSON: %s", client.executeRunInput.Input)
	}
}

func TestRunCommandCallsClientWithExplicitRevision(t *testing.T) {
	t.Parallel()

	client := &fakeRuntimeClient{run: RunResponse{
		RunID: "run-1", AgentName: "release-notes-helper", Status: "running",
	}}
	var out bytes.Buffer
	cmd := NewRunCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper:11111111-1111-4111-8111-111111111111"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.executeAgent != "release-notes-helper:11111111-1111-4111-8111-111111111111" {
		t.Fatalf("execute agent: got %q", client.executeAgent)
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
	requireCommand(t, cmd, "run")
	requireCommand(t, cmd, "stop")
}

type fakeRuntimeClient struct {
	executeAgent    string
	executeInputs   map[string]string
	executeRunInput RunInput
	stopRequest     StopRequest
	run             RunResponse
	err             error
}

func (f *fakeRuntimeClient) Execute(_ context.Context, agentName string, inputs map[string]string) (RunResponse, error) {
	f.executeAgent = agentName
	f.executeInputs = inputs
	if f.err != nil {
		return RunResponse{}, f.err
	}

	return f.run, nil
}

func (f *fakeRuntimeClient) ExecuteWithInput(_ context.Context, agentName string, input RunInput) (RunResponse, error) {
	f.executeAgent = agentName
	f.executeRunInput = input
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
