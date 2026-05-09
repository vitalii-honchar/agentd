package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func TestPSCommandListsActiveRuns(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)
	client := &fakeQueryClient{runsResponse: RunListResponse{Runs: []RunSummary{{
		RunID:     "run-1",
		AgentName: "hacker-news-builder-brief",
		Status:    "running",
		Trigger:   "manual",
		StartedAt: &started,
	}}}}
	var out bytes.Buffer
	cmd := NewPSCommand(client, NewOutput(config.OutputText, &out))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.listRunsIncludeAll {
		t.Fatal("include all: got true want false")
	}
	if !strings.Contains(out.String(), "hacker-news-builder-brief") || !strings.Contains(out.String(), "running") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestPSCommandListsAllRuns(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, 5, 8, 9, 2, 0, 0, time.UTC)
	client := &fakeQueryClient{runsResponse: RunListResponse{Runs: []RunSummary{{
		RunID:       "run-2",
		AgentName:   "cybersecurity-reddit-watch",
		Status:      "completed",
		Trigger:     "schedule",
		CompletedAt: &completed,
	}}}}
	var out bytes.Buffer
	cmd := NewPSCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"-a"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !client.listRunsIncludeAll {
		t.Fatal("include all: got false want true")
	}
	if !strings.Contains(out.String(), "cybersecurity-reddit-watch") || !strings.Contains(out.String(), "completed") {
		t.Fatalf("output: %q", out.String())
	}
}
