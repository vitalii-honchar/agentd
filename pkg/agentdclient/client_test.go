package agentdclient

import (
	"context"
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
