package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func TestLogsCommandReadsRunIDScopedLogs(t *testing.T) {
	t.Parallel()

	runID := "11111111-1111-4111-8111-111111111111"
	client := &fakeQueryClient{logsResponse: LogsResponse{
		RunID: runID,
		Entries: []LogEntry{{
			Timestamp: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			RunID:     runID,
			Action:    "react.step",
			Message:   "react step completed",
			Line:      "react step completed",
		}},
	}}
	var out bytes.Buffer
	cmd := NewLogsCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{runID, "--tail", "5"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.logsRequest.RunID != runID {
		t.Fatalf("run id: got %q want %q", client.logsRequest.RunID, runID)
	}
	if client.logsRequest.AgentName != "" {
		t.Fatalf("agent name should not be passed for run-scoped logs: %q", client.logsRequest.AgentName)
	}
	if client.logsRequest.Tail != 5 {
		t.Fatalf("tail: got %d want 5", client.logsRequest.Tail)
	}
	if !strings.Contains(out.String(), "react.step") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestLogsCommandRejectsAgentNameArgument(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{}
	var out bytes.Buffer
	cmd := NewLogsCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if client.logsRequest.AgentName != "" || client.logsRequest.RunID != "" {
		t.Fatalf("logs should not be queried for agent name argument: %#v", client.logsRequest)
	}
}
