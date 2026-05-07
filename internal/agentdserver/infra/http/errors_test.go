package http

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func TestAPIErrorResponsesUseConsistentEnvelope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		server     *Server
		request    *stdhttp.Request
		wantStatus int
		wantCode   string
	}{
		{
			name:       "inspect not found",
			server:     NewServer(Config{}, WithInspectUseCase(&fakeInspectUseCase{err: domain.ErrNotFound})),
			request:    httptest.NewRequest(stdhttp.MethodGet, "/v1/agents/missing", nil),
			wantStatus: stdhttp.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "execute conflict",
			server:     NewServer(Config{}, WithExecuteUseCase(&fakeExecuteUseCase{err: domain.ErrRunAlreadyActive})),
			request:    httptest.NewRequest(stdhttp.MethodPost, "/v1/agents/release-notes-helper/runs", nil),
			wantStatus: stdhttp.StatusConflict,
			wantCode:   "conflict",
		},
		{
			name:       "logs invalid query",
			server:     NewServer(Config{}, WithLogsUseCase(&fakeLogsUseCase{})),
			request:    httptest.NewRequest(stdhttp.MethodGet, "/v1/agents/release-notes-helper/logs?tail=0", nil),
			wantStatus: stdhttp.StatusBadRequest,
			wantCode:   "invalid_query",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			tt.server.Handler().ServeHTTP(response, tt.request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status: got %d want %d body %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("content type: got %q want application/json", contentType)
			}
			var body model.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error.Code != tt.wantCode || body.Error.Message == "" {
				t.Fatalf("error body: %#v", body.Error)
			}
		})
	}
}
