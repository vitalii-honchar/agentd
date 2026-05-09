package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestRunQueryHandlerListsActiveRuns(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)
	useCase := &fakeRunListUseCase{runs: []domain.AgentRun{{
		ID:        "11111111-1111-4111-8111-111111111111",
		AgentName: "hacker-news-builder-brief",
		Status:    domain.AgentRunStatusRunning,
		Trigger:   domain.RunTriggerManual,
		StartedAt: &started,
	}}}
	server := NewServer(Config{}, WithRunListUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/runs", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	if useCase.includeAll {
		t.Fatal("includeAll: got true want false")
	}
	var body model.RunListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Runs) != 1 || body.Runs[0].RunID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("runs: %#v", body.Runs)
	}
	if body.Runs[0].Trigger != string(domain.RunTriggerManual) {
		t.Fatalf("trigger: got %q", body.Runs[0].Trigger)
	}
}

func TestRunQueryHandlerListsAllRuns(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, 5, 8, 9, 2, 0, 0, time.UTC)
	useCase := &fakeRunListUseCase{runs: []domain.AgentRun{{
		ID:          "22222222-2222-4222-8222-222222222222",
		AgentName:   "hacker-news-builder-brief",
		Status:      domain.AgentRunStatusCompleted,
		Trigger:     domain.RunTriggerSchedule,
		CompletedAt: &completed,
	}}}
	server := NewServer(Config{}, WithRunListUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/runs?all=true", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	if !useCase.includeAll {
		t.Fatal("includeAll: got false want true")
	}
	var body model.RunListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Runs) != 1 || body.Runs[0].Status != string(domain.AgentRunStatusCompleted) {
		t.Fatalf("runs: %#v", body.Runs)
	}
}

type fakeRunListUseCase struct {
	includeAll bool
	runs       []domain.AgentRun
	err        error
}

func (f *fakeRunListUseCase) ListRuns(_ context.Context, includeAll bool) ([]domain.AgentRun, error) {
	f.includeAll = includeAll
	if f.err != nil {
		return nil, f.err
	}

	return f.runs, nil
}
