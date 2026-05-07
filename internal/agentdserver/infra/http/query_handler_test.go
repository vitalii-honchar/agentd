package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agentd/internal/agentdserver/app"
	applogs "agentd/internal/agentdserver/app/logs"
	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func TestListHandlerReturnsAgents(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithListUseCase(&fakeListUseCase{agents: []domain.Agent{
		testHTTPAgent("daily-pr-review"),
		testHTTPAgent("release-notes-helper"),
	}}))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/v1/agents", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.ListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Agents) != 2 || body.Agents[0].Name != "daily-pr-review" {
		t.Fatalf("agents: %#v", body.Agents)
	}
}

func TestInspectHandlerReturnsAgent(t *testing.T) {
	t.Parallel()

	inspect := &fakeInspectUseCase{agent: testHTTPAgent("release-notes-helper")}
	server := NewServer(Config{}, WithInspectUseCase(inspect))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/v1/agents/release-notes-helper", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.AgentDetail
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "release-notes-helper" || inspect.name != "release-notes-helper" {
		t.Fatalf("body=%#v inspect=%q", body, inspect.name)
	}
}

func TestInspectHandlerNotFound(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithInspectUseCase(&fakeInspectUseCase{err: domain.ErrNotFound}))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/v1/agents/missing", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusNotFound {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusNotFound)
	}
}

func TestLogsHandlerReturnsEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)
	logsUseCase := &fakeLogsUseCase{result: applogs.Result{
		Agent: testHTTPAgent("release-notes-helper"),
		Run: domain.AgentRun{
			ID:        "run-1",
			AgentName: "release-notes-helper",
			Status:    domain.AgentRunStatusCompleted,
		},
		Entries: []app.LogEntry{{
			Timestamp: now,
			RunID:     "run-1",
			Line:      "completed",
		}},
	}}
	server := NewServer(Config{}, WithLogsUseCase(logsUseCase))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		stdhttp.MethodGet,
		"/v1/agents/release-notes-helper/logs?run_id=run-1&tail=20",
		nil,
	)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.LogsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AgentName != "release-notes-helper" || body.RunID != "run-1" || body.Entries[0].Line != "completed" {
		t.Fatalf("body: %#v", body)
	}
	if logsUseCase.query.RunID != "run-1" || logsUseCase.query.Tail != 20 {
		t.Fatalf("query: %#v", logsUseCase.query)
	}
}

func TestLogsHandlerRejectsInvalidTail(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithLogsUseCase(&fakeLogsUseCase{}))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/v1/agents/release-notes-helper/logs?tail=bad", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusBadRequest)
	}
}

type fakeListUseCase struct {
	agents []domain.Agent
	err    error
}

func (f *fakeListUseCase) List(context.Context) ([]domain.Agent, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.agents, nil
}

type fakeInspectUseCase struct {
	name  string
	agent domain.Agent
	err   error
}

func (f *fakeInspectUseCase) Inspect(_ context.Context, name string) (domain.Agent, error) {
	f.name = name
	if f.err != nil {
		return domain.Agent{}, f.err
	}

	return f.agent, nil
}

type fakeLogsUseCase struct {
	query  applogs.Query
	result applogs.Result
	err    error
}

func (f *fakeLogsUseCase) Read(_ context.Context, query applogs.Query) (applogs.Result, error) {
	f.query = query
	if f.err != nil {
		return applogs.Result{}, f.err
	}

	return f.result, nil
}

func testHTTPAgent(name string) domain.Agent {
	return domain.Agent{
		Name:      name,
		Revision:  "rev-1",
		Enabled:   true,
		Status:    domain.AgentStatusActive,
		Vendor:    domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:  domain.Schedule{Type: domain.ScheduleTypeManual},
		LastRunID: "run-1",
	}
}
