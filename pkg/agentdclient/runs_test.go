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

func TestClientExecuteWithStructuredInput(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(RunSummary{
			RunID:     "run-1",
			AgentName: "contracted-agent",
			Status:    "running",
		})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	run, err := client.ExecuteWithInput(context.Background(), "contracted-agent", RunInput{
		Input: json.RawMessage(`{"topic":"agentd","limit":3}`),
	})
	if err != nil {
		t.Fatalf("ExecuteWithInput: %v", err)
	}
	if run.RunID != "run-1" {
		t.Fatalf("run: %#v", run)
	}
	if gotPath != "/v1/agents/contracted-agent/runs" {
		t.Fatalf("path: got %q", gotPath)
	}
	input, ok := gotBody["input"].(map[string]any)
	if !ok {
		t.Fatalf("request input: %#v", gotBody)
	}
	if input["topic"] != "agentd" || input["limit"] != float64(3) {
		t.Fatalf("request input: %#v", input)
	}
	if _, ok := gotBody["legacy_inputs"]; ok {
		t.Fatalf("structured request should not include legacy_inputs: %#v", gotBody)
	}
}

func TestClientExecuteWithLegacyInputs(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(RunSummary{
			RunID:     "run-1",
			AgentName: "legacy-agent",
			Status:    "running",
		})
	}))
	t.Cleanup(server.Close)
	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.ExecuteWithInput(context.Background(), "legacy-agent", RunInput{
		LegacyInputs: map[string]string{"url": "https://example.com"},
	}); err != nil {
		t.Fatalf("ExecuteWithInput: %v", err)
	}
	legacyInputs, ok := gotBody["legacy_inputs"].(map[string]any)
	if !ok {
		t.Fatalf("request legacy_inputs: %#v", gotBody)
	}
	if legacyInputs["url"] != "https://example.com" {
		t.Fatalf("request legacy_inputs: %#v", legacyInputs)
	}
}
