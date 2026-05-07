package http

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	appagent "agentd/internal/agentdserver/app/agent"
	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func TestApplyHandlerCreated(t *testing.T) {
	t.Parallel()

	useCase := &fakeApplyUseCase{
		result: appagent.ApplyResult{
			Outcome: appagent.ApplyOutcomeCreated,
			Agent: domain.Agent{
				Name:     "release-notes-helper",
				Revision: "rev-1",
				Enabled:  true,
				Status:   domain.AgentStatusActive,
				Vendor:   domain.Vendor{Name: "openai", Model: "gpt-5"},
				Schedule: domain.Schedule{Type: domain.ScheduleTypeManual},
			},
		},
	}
	server := NewServer(Config{}, WithApplyUseCase(useCase))

	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, jsonRequest(t, map[string]string{
		"source_path": "agent.md",
		"markdown":    "---\nname: release-notes-helper\n---\nprompt",
	}))

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.ApplyResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Outcome != string(appagent.ApplyOutcomeCreated) {
		t.Fatalf("outcome: got %q", body.Outcome)
	}
	if body.Agent.Name != "release-notes-helper" {
		t.Fatalf("agent name: got %q", body.Agent.Name)
	}
	if useCase.request.SourcePath != "agent.md" {
		t.Fatalf("source path: got %q", useCase.request.SourcePath)
	}
}

func TestApplyHandlerValidationError(t *testing.T) {
	t.Parallel()

	useCase := &fakeApplyUseCase{
		err: domain.NewValidationError([]domain.ValidationIssue{{
			Field:   "name",
			Message: "is required",
		}}),
	}
	server := NewServer(Config{}, WithApplyUseCase(useCase))

	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, jsonRequest(t, map[string]string{
		"source_path": "bad.md",
		"markdown":    "---\nname: ''\n---\n",
	}))

	if response.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusBadRequest, response.Body.String())
	}
	var body model.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "invalid_definition" {
		t.Fatalf("error code: got %q", body.Error.Code)
	}
	if len(body.Error.Fields) != 1 || body.Error.Fields[0].Path != "name" {
		t.Fatalf("fields: %#v", body.Error.Fields)
	}
}

func TestApplyHandlerInvalidJSON(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithApplyUseCase(&fakeApplyUseCase{}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/v1/agents/apply", bytes.NewBufferString("{"))
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusBadRequest)
	}
}

type fakeApplyUseCase struct {
	request appagent.ApplyRequest
	result  appagent.ApplyResult
	err     error
}

func (f *fakeApplyUseCase) Apply(
	_ context.Context,
	request appagent.ApplyRequest,
) (appagent.ApplyResult, error) {
	f.request = request
	if f.err != nil {
		return appagent.ApplyResult{}, f.err
	}

	return f.result, nil
}

func jsonRequest(t *testing.T, body any) *stdhttp.Request {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(stdhttp.MethodPost, "/v1/agents/apply", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")

	return request
}
