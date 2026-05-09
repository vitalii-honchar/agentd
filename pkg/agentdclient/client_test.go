package agentdclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDecodesDaemonErrorResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"agent_not_found","message":"agent missing"}}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = client.Health(context.Background())
	if err == nil {
		t.Fatal("Health error is nil")
	}
	var daemonErr *Error
	if !errors.As(err, &daemonErr) {
		t.Fatalf("error type: got %T want *Error", err)
	}
	if daemonErr.Code != ErrorCodeAgentNotFound {
		t.Fatalf("code: got %q want %q", daemonErr.Code, ErrorCodeAgentNotFound)
	}
	if daemonErr.Message != "agent missing" {
		t.Fatalf("message: got %q", daemonErr.Message)
	}
	if daemonErr.HTTPStatus != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", daemonErr.HTTPStatus, http.StatusNotFound)
	}
}

func TestClientReturnsDaemonUnavailableForTransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()

	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = client.Health(context.Background())
	if err == nil {
		t.Fatal("Health error is nil")
	}
	var daemonErr *Error
	if !errors.As(err, &daemonErr) {
		t.Fatalf("error type: got %T want *Error", err)
	}
	if daemonErr.Code != ErrorCodeDaemonUnavailable {
		t.Fatalf("code: got %q want %q", daemonErr.Code, ErrorCodeDaemonUnavailable)
	}
}

func TestClientLogsUsesRunScopedEndpoint(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(LogsResult{
			RunID: "run-1",
			Entries: []LogEntry{
				{RunID: "run-1", Action: "react.step", Line: `{"action":"react.step"}`},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := client.Logs(context.Background(), LogsQuery{RunID: "run-1", Tail: 20})
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if gotPath != "/v1/runs/run-1/logs?tail=20" {
		t.Fatalf("path: got %q want %q", gotPath, "/v1/runs/run-1/logs?tail=20")
	}
	if result.RunID != "run-1" {
		t.Fatalf("run id: got %q want %q", result.RunID, "run-1")
	}
	if len(result.Entries) != 1 || result.Entries[0].RunID != "run-1" {
		t.Fatalf("entries: got %#v", result.Entries)
	}
}

func TestClientLogsRequiresRunID(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	t.Cleanup(server.Close)

	client, err := New(Config{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = client.Logs(context.Background(), LogsQuery{AgentName: "release-notes-helper"})
	if err == nil {
		t.Fatal("Logs error is nil")
	}
	if called {
		t.Fatal("server was called for agent-name-only logs query")
	}
}
