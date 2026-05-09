package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestExecuteHandlerAccepted(t *testing.T) {
	t.Parallel()

	execute := &fakeExecuteUseCase{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		Status:    domain.AgentRunStatusRunning,
	}}
	server := NewServer(Config{}, WithExecuteUseCase(execute))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodPost, "/v1/agents/release-notes-helper/runs", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusAccepted {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusAccepted, response.Body.String())
	}
	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RunID != "run-1" || body.AgentName != "release-notes-helper" {
		t.Fatalf("response: %#v", body)
	}
	if execute.agentName != "release-notes-helper" {
		t.Fatalf("agent name: got %q", execute.agentName)
	}
}

func TestExecuteHandlerPassesInputs(t *testing.T) {
	t.Parallel()

	execute := &fakeExecuteUseCase{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: "website-snapshot-analyst",
		Status:    domain.AgentRunStatusRunning,
	}}
	server := NewServer(Config{}, WithExecuteUseCase(execute))
	response := httptest.NewRecorder()
	request := localRequest(
		stdhttp.MethodPost,
		"/v1/agents/website-snapshot-analyst/runs",
		strings.NewReader(`{"inputs":{"url":"https://example.com"}}`),
	)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusAccepted {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusAccepted, response.Body.String())
	}
	if execute.inputs["url"] != "https://example.com" {
		t.Fatalf("inputs: %#v", execute.inputs)
	}
}

func TestExecuteHandlerPassesExplicitRevisionSelector(t *testing.T) {
	t.Parallel()

	execute := &fakeExecuteUseCase{run: domain.AgentRun{
		ID:            "run-1",
		AgentName:     "release-notes-helper",
		AgentRevision: "11111111-1111-4111-8111-111111111111",
		Status:        domain.AgentRunStatusRunning,
	}}
	server := NewServer(Config{}, WithExecuteUseCase(execute))
	response := httptest.NewRecorder()
	request := localRequest(
		stdhttp.MethodPost,
		"/v1/agents/release-notes-helper:11111111-1111-4111-8111-111111111111/runs",
		nil,
	)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusAccepted {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusAccepted, response.Body.String())
	}
	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AgentRevision != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("agent revision: got %q", body.AgentRevision)
	}
	if execute.agentName != "release-notes-helper:11111111-1111-4111-8111-111111111111" {
		t.Fatalf("agent selector: got %q", execute.agentName)
	}
}

func TestExecuteHandlerRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithExecuteUseCase(&fakeExecuteUseCase{}))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodPost, "/v1/agents/website-snapshot-analyst/runs", strings.NewReader(`{`))

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusBadRequest)
	}
}

func TestExecuteHandlerConflict(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithExecuteUseCase(&fakeExecuteUseCase{err: domain.ErrRunAlreadyActive}))
	response := httptest.NewRecorder()
	request := localRequest(stdhttp.MethodPost, "/v1/agents/release-notes-helper/runs", nil)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusConflict {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusConflict)
	}
}

func TestStopHandlerAccepted(t *testing.T) {
	t.Parallel()

	stop := &fakeStopUseCase{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		Status:    domain.AgentRunStatusStopping,
	}}
	server := NewServer(Config{}, WithStopUseCase(stop))
	response := httptest.NewRecorder()
	request := localRequest(
		stdhttp.MethodPost,
		"/v1/agents/release-notes-helper/runs/run-1/stop",
		nil,
	)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusAccepted {
		t.Fatalf("status: got %d want %d body %s", response.Code, stdhttp.StatusAccepted, response.Body.String())
	}
	if stop.agentName != "release-notes-helper" || stop.runID != "run-1" {
		t.Fatalf("stop args: %q %q", stop.agentName, stop.runID)
	}
}

func TestStopHandlerNotFound(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{}, WithStopUseCase(&fakeStopUseCase{err: domain.ErrNotFound}))
	response := httptest.NewRecorder()
	request := localRequest(
		stdhttp.MethodPost,
		"/v1/agents/missing/runs/run-1/stop",
		nil,
	)

	server.Handler().ServeHTTP(response, request)

	if response.Code != stdhttp.StatusNotFound {
		t.Fatalf("status: got %d want %d", response.Code, stdhttp.StatusNotFound)
	}
}

type fakeExecuteUseCase struct {
	agentName string
	inputs    map[string]string
	run       domain.AgentRun
	err       error
}

func (f *fakeExecuteUseCase) Execute(_ context.Context, agentName string, inputs map[string]string) (domain.AgentRun, error) {
	f.agentName = agentName
	f.inputs = inputs
	if f.err != nil {
		return domain.AgentRun{}, f.err
	}

	return f.run, nil
}

type fakeStopUseCase struct {
	agentName string
	runID     string
	run       domain.AgentRun
	err       error
}

func (f *fakeStopUseCase) Stop(_ context.Context, agentName, runID string) (domain.AgentRun, error) {
	f.agentName = agentName
	f.runID = runID
	if f.err != nil {
		return domain.AgentRun{}, f.err
	}

	return f.run, nil
}
