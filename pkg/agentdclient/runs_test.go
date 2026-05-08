package agentdclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListRunsUsesActiveEndpointByDefault(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_ = json.NewEncoder(w).Encode(RunListResponse{Runs: []RunSummary{{
			RunID:     "run-1",
			AgentName: "hacker-news-builder-brief",
			Status:    "running",
			Trigger:   "manual",
		}}})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runs, err := client.ListRuns(context.Background(), false)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if gotPath != "/v1/runs" {
		t.Fatalf("path: got %q want /v1/runs", gotPath)
	}
	if len(runs) != 1 || runs[0].RunID != "run-1" {
		t.Fatalf("runs: %#v", runs)
	}
}

func TestClientListRunsCanIncludeAllRuns(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_ = json.NewEncoder(w).Encode(RunListResponse{})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.ListRuns(context.Background(), true); err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if gotPath != "/v1/runs?all=true" {
		t.Fatalf("path: got %q want /v1/runs?all=true", gotPath)
	}
}
