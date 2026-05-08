package http

import (
	"encoding/json"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{
		Address:      "127.0.0.1:0",
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})

	request := localRequest(stdhttp.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type: got %q want application/json", contentType)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body: got %q want ok", body["status"])
	}
}

func localRequest(method, target string, body io.Reader) *stdhttp.Request {
	request := httptest.NewRequest(method, target, body)
	request.RemoteAddr = "127.0.0.1:12345"

	return request
}
