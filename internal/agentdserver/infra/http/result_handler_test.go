package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestResultHandlerListsAgentResults(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	useCase := &fakeResultUseCase{agentResults: []appresult.RunResult{{
		RunID:         "run-1",
		AgentName:     "hacker-news-builder-brief",
		Status:        domain.AgentRunStatusCompleted,
		Trigger:       domain.RunTriggerSchedule,
		CompletedAt:   &completed,
		ResultSummary: "top HN stories",
	}}}
	server := NewServer(Config{}, WithResultUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/agents/hacker-news-builder-brief/results", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	if useCase.agentName != "hacker-news-builder-brief" {
		t.Fatalf("agent name: got %q", useCase.agentName)
	}
	var body model.AgentResultsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Results) != 1 || body.Results[0].ResultSummary != "top HN stories" {
		t.Fatalf("body: %#v", body)
	}
}

func TestResultHandlerReturnsFullRunResult(t *testing.T) {
	t.Parallel()

	useCase := &fakeResultUseCase{runResult: appresult.RunResult{
		RunID:         "run-2",
		AgentName:     "website-snapshot-analyst",
		Status:        domain.AgentRunStatusFailed,
		Trigger:       domain.RunTriggerManual,
		Result:        "full failed result",
		ResultSummary: "failed",
		Failure:       &appresult.Failure{Code: "tool_failed", Message: "tool failed"},
	}}
	server := NewServer(Config{}, WithResultUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/runs/run-2/result", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	if useCase.runID != "run-2" {
		t.Fatalf("run id: got %q", useCase.runID)
	}
	var body model.RunResult
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Result != "full failed result" || body.Failure == nil {
		t.Fatalf("body: %#v", body)
	}
}

func TestResultHandlerReturnsStructuredJSONResult(t *testing.T) {
	t.Parallel()

	useCase := &fakeResultUseCase{runResult: appresult.RunResult{
		RunID:         "run-json",
		AgentName:     "contracted-agent",
		Status:        domain.AgentRunStatusCompleted,
		Trigger:       domain.RunTriggerManual,
		ResultFormat:  domain.ResultFormatJSON,
		Result:        `{"summary":"done","score":0.91}`,
		ResultJSON:    json.RawMessage(`{"summary":"done","score":0.91}`),
		ResultSummary: "done",
	}}
	server := NewServer(Config{}, WithResultUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/runs/run-json/result", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.RunResult
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ResultFormat != string(domain.ResultFormatJSON) {
		t.Fatalf("result format: got %q", body.ResultFormat)
	}
	var resultJSON map[string]any
	if err := json.Unmarshal(body.ResultJSON, &resultJSON); err != nil {
		t.Fatalf("result_json is not JSON object: %v raw=%s", err, body.ResultJSON)
	}
	if resultJSON["summary"] != "done" || resultJSON["score"] != 0.91 {
		t.Fatalf("result_json: %#v", resultJSON)
	}
}

func TestResultHandlerReturnsLegacyTextResult(t *testing.T) {
	t.Parallel()

	useCase := &fakeResultUseCase{runResult: appresult.RunResult{
		RunID:         "run-text",
		AgentName:     "legacy-agent",
		Status:        domain.AgentRunStatusCompleted,
		Trigger:       domain.RunTriggerManual,
		ResultFormat:  domain.ResultFormatText,
		Result:        "plain result",
		ResultSummary: "plain result",
	}}
	server := NewServer(Config{}, WithResultUseCase(useCase))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodGet, "/v1/runs/run-text/result", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusOK, response.Body.String())
	}
	var body model.RunResult
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ResultFormat != string(domain.ResultFormatText) {
		t.Fatalf("result format: got %q", body.ResultFormat)
	}
	if body.Result != "plain result" {
		t.Fatalf("result: got %q", body.Result)
	}
	if len(body.ResultJSON) != 0 {
		t.Fatalf("legacy result should not include result_json: %s", body.ResultJSON)
	}
}

type fakeResultUseCase struct {
	agentName    string
	runID        string
	agentResults []appresult.RunResult
	runResult    appresult.RunResult
	err          error
}

func (f *fakeResultUseCase) ResultsByAgent(_ context.Context, agentName string) ([]appresult.RunResult, error) {
	f.agentName = agentName
	if f.err != nil {
		return nil, f.err
	}

	return f.agentResults, nil
}

func (f *fakeResultUseCase) ResultByRunID(_ context.Context, runID string) (appresult.RunResult, error) {
	f.runID = runID
	if f.err != nil {
		return appresult.RunResult{}, f.err
	}

	return f.runResult, nil
}
