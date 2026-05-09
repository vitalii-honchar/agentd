package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
	infraruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/runtime"
)

func TestRecoveryResultPersistsAcrossRuntimeDBReopen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsDB := openSettingsDB(t, filepath.Join(dir, "settings.db"))
	agentRepo, err := repository.NewAgentRepository(settingsDB)
	if err != nil {
		t.Fatalf("NewAgentRepository: %v", err)
	}
	agent := domain.Agent{
		Name:               "hacker-news-builder-brief",
		Revision:           "rev-1",
		DefinitionSource:   "examples/hacker-news-builder-brief/hacker-news-builder-brief.md",
		DefinitionMarkdown: "definition",
		Prompt:             "prompt",
		Enabled:            true,
		Vendor:             domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:           domain.Schedule{Type: domain.ScheduleTypeCron, Expression: "0 8 * * *"},
		Status:             domain.AgentStatusActive,
	}
	if err := agentRepo.Save(context.Background(), agent, nil, nil); err != nil {
		t.Fatalf("Save: %v", err)
	}

	runtimeDir := filepath.Join(dir, "runtime")
	runtimeDBs := openRuntimeDBs(t, runtimeDir)
	if err := runtimeDBs.EnsureAgent(context.Background(), agent.Name); err != nil {
		t.Fatalf("EnsureAgent: %v", err)
	}
	now := time.Now().UTC()
	run := domain.AgentRun{
		ID:            "11111111-1111-1111-1111-111111111111",
		AgentName:     agent.Name,
		AgentRevision: agent.Revision,
		Trigger:       domain.RunTriggerSchedule,
		Status:        domain.AgentRunStatusCompleted,
		StartedAt:     &now,
		CompletedAt:   &now,
		Result:        "persistent HN result",
		ResultSummary: "persistent HN result",
	}
	if err := runtimeDBs.Runs(agent.Name).Create(context.Background(), run); err != nil {
		t.Fatalf("Create run: %v", err)
	}
	if err := runtimeDBs.Close(context.Background()); err != nil {
		t.Fatalf("Close runtime DBs: %v", err)
	}

	reopened := openRuntimeDBs(t, runtimeDir)
	resultUC, err := appresult.NewUseCase(agentRepo, reopened)
	if err != nil {
		t.Fatalf("NewResultUseCase: %v", err)
	}
	result, err := resultUC.ResultByRunID(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ResultByRunID: %v", err)
	}
	if result.Result != run.Result {
		t.Fatalf("result: got %q want %q", result.Result, run.Result)
	}
}

func TestManagerRecoveryInterruptsActiveToolProcess(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, instantE2EProvider{})
	stack.manager.SetToolExecutor(infraruntime.NewProcessToolExecutor(5 * time.Second))

	exampleDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(exampleDir, "tools"), 0o755); err != nil {
		t.Fatalf("Mkdir tools: %v", err)
	}
	toolPath := filepath.Join(exampleDir, "tools", "slow.sh")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n(sleep 1; echo late > marker.txt) & wait\n"), 0o700); err != nil {
		t.Fatalf("WriteFile tool: %v", err)
	}
	postApply(t, stack.server, filepath.Join(exampleDir, "tool-agent.md"), toolRecoveryDefinition())
	runResponse := postRun(t, stack.server, "tool-recovery-agent")
	waitForEventType(t, stack.runtimeDBs, "tool-recovery-agent", runResponse.RunID, domain.RunActionToolExecuteStart)
	run, err := stack.runtimeDBs.Runs("tool-recovery-agent").FindByID(context.Background(), runResponse.RunID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	recovery, err := stack.manager.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recovery.InterruptedRuns) != 1 {
		t.Fatalf("interrupted runs: got %d want 1", len(recovery.InterruptedRuns))
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "tool-recovery-agent", runResponse.RunID, domain.AgentRunStatusInterrupted)
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(run.WorkDir, "marker.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tool child process was not interrupted; marker stat error: %v", err)
	}
}

func TestContractedRunRecoveryAndObservabilityEvents(t *testing.T) {
	t.Parallel()

	provider := &countingE2EProvider{}
	stack := newRuntimeStackWithProvider(t, provider)
	postApply(t, stack.server, "contracted-agent.md", contractedE2EDefinition())

	response := postRunRaw(t, stack.server, "contracted-agent", `{"input":{"topic":"agentd"}}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("run status: got %d body %s", response.Code, response.Body.String())
	}
	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "contracted-agent", body.RunID, domain.AgentRunStatusCompleted)
	waitForEventType(t, stack.runtimeDBs, "contracted-agent", body.RunID, domain.RunActionContractInputValidated)
	waitForEventType(t, stack.runtimeDBs, "contracted-agent", body.RunID, domain.RunActionOutputFinalizeDone)

	result, err := stack.runtimeDBs.Runs("contracted-agent").FindByID(context.Background(), body.RunID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if result.ResultFormat != domain.ResultFormatJSON || result.ContractOutputSchemaDigest == "" {
		t.Fatalf("contracted run metadata: %#v", result)
	}
}

func openSettingsDB(t *testing.T, path string) *db.DB {
	t.Helper()

	settingsDB, err := db.New("settings", db.Config{
		Path:         path,
		MaxOpenConns: 1,
		Pragmas:      db.PragmasSettings,
	})
	if err != nil {
		t.Fatalf("New settings DB: %v", err)
	}
	t.Cleanup(func() {
		if err := settingsDB.Stop(context.Background()); err != nil {
			t.Fatalf("Stop settings DB: %v", err)
		}
	})
	if err := settingsDB.Start(context.Background()); err != nil {
		t.Fatalf("Start settings DB: %v", err)
	}

	return settingsDB
}

func openRuntimeDBs(t *testing.T, dir string) *repository.RuntimeDBManager {
	t.Helper()

	runtimeDBs, err := repository.NewRuntimeDBManager(dir, 1)
	if err != nil {
		t.Fatalf("NewRuntimeDBManager: %v", err)
	}
	t.Cleanup(func() {
		if err := runtimeDBs.Close(context.Background()); err != nil {
			t.Fatalf("Close runtime DBs: %v", err)
		}
	})

	return runtimeDBs
}

func waitForEventType(
	t *testing.T,
	runtimeDBs *repository.RuntimeDBManager,
	agentName string,
	runID string,
	eventType string,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events, err := runtimeDBs.Events(agentName).ListByRun(context.Background(), runID, 20)
		if err == nil {
			for _, event := range events {
				if event.EventType == eventType {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("event %q was not recorded", eventType)
}

type instantE2EProvider struct{}

func (instantE2EProvider) Name() string {
	return "openai"
}

func (instantE2EProvider) Execute(context.Context, appruntime.ProviderRequest) (appruntime.ProviderResponse, error) {
	return appruntime.ProviderResponse{RequestID: "request-1", Output: "provider output"}, nil
}

func postRunWithInputs(
	t *testing.T,
	server *daemonhttp.Server,
	agentName string,
	inputs map[string]string,
) model.RunResponse {
	t.Helper()

	payload, err := json.Marshal(model.ExecuteRequest{Inputs: inputs})
	if err != nil {
		t.Fatalf("Marshal execute request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/agents/"+agentName+"/runs", bytes.NewReader(payload))
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("run status: got %d body %s", response.Code, response.Body.String())
	}

	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode run response: %v", err)
	}

	return body
}

func toolRecoveryDefinition() string {
	return `---
name: tool-recovery-agent
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools:
  - name: slow
    kind: local_tool
    command: tools/slow.sh
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
Run the declared slow tool before answering.`
}
