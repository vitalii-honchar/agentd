package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func TestResultCommandShowsAgentResultTable(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	client := &fakeQueryClient{agentResultsResponse: AgentResultsResponse{
		AgentName: "hacker-news-builder-brief",
		Results: []RunResult{{
			RunSummary:    RunSummary{RunID: "run-1", AgentName: "hacker-news-builder-brief", Status: "completed", CompletedAt: &completed},
			ResultSummary: strings.Repeat("important ", 20),
		}},
	}}
	var out bytes.Buffer
	cmd := NewResultCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"hacker-news-builder-brief"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.resultAgentName != "hacker-news-builder-brief" {
		t.Fatalf("agent name: got %q", client.resultAgentName)
	}
	if !strings.Contains(out.String(), "run-1") || !strings.Contains(out.String(), "...") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestResultCommandShowsFullRunResultJSON(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{runResult: RunResult{
		RunSummary: RunSummary{RunID: "11111111-1111-4111-8111-111111111111", AgentName: "website-snapshot-analyst", Status: "failed"},
		Result:     "full result",
		Failure:    &Failure{Code: "tool_failed", Message: "tool failed"},
	}}
	var out bytes.Buffer
	cmd := NewResultCommand(client, NewOutput(config.OutputJSON, &out))
	cmd.SetArgs([]string{"11111111-1111-4111-8111-111111111111"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute error is nil")
	} else if ExitCode(err) != 5 {
		t.Fatalf("exit code: got %d want 5", ExitCode(err))
	}
	if client.resultRunID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("run id: got %q", client.resultRunID)
	}
	if !strings.Contains(out.String(), `"result": "full result"`) {
		t.Fatalf("output: %q", out.String())
	}
}
