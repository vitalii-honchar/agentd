package agentdclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientResultsByAgent(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_ = json.NewEncoder(w).Encode(AgentResultsResponse{
			AgentName: "hacker-news-builder-brief",
			Results: []RunResult{{
				RunSummary:    RunSummary{RunID: "run-1", AgentName: "hacker-news-builder-brief", Status: "completed"},
				ResultSummary: "HN summary",
			}},
		})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	results, err := client.ResultsByAgent(context.Background(), "hacker-news-builder-brief")
	if err != nil {
		t.Fatalf("ResultsByAgent: %v", err)
	}
	if gotPath != "/v1/agents/hacker-news-builder-brief/results" {
		t.Fatalf("path: got %q", gotPath)
	}
	if len(results) != 1 || results[0].ResultSummary != "HN summary" {
		t.Fatalf("results: %#v", results)
	}
}

func TestClientResultByRunID(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_ = json.NewEncoder(w).Encode(RunResult{
			RunSummary: RunSummary{RunID: "run-2", AgentName: "website-snapshot-analyst", Status: "failed"},
			Result:     "full result",
			Failure:    &Failure{Code: "tool_failed", Message: "tool failed"},
		})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := client.ResultByRunID(context.Background(), "run-2")
	if err != nil {
		t.Fatalf("ResultByRunID: %v", err)
	}
	if gotPath != "/v1/runs/run-2/result" {
		t.Fatalf("path: got %q", gotPath)
	}
	if result.Result != "full result" || result.Failure == nil {
		t.Fatalf("result: %#v", result)
	}
}
