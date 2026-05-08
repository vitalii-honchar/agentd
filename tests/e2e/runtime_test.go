package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	applogs "github.com/vitalii-honchar/agentd/internal/agentdserver/app/logs"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
	runlogs "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/logs"
	infraruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/runtime"
)

func TestRuntimeConcurrencyStopAndRecovery(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, blockingE2EProvider{})
	postApply(t, stack.server, "agent-a.md", runtimeDefinition("agent-a"))
	postApply(t, stack.server, "agent-b.md", runtimeDefinition("agent-b"))

	runA := postRun(t, stack.server, "agent-a")
	runB := postRun(t, stack.server, "agent-b")
	if runA.RunID == runB.RunID {
		t.Fatal("run IDs should differ")
	}

	active, err := stack.manager.ActiveRuns(context.Background())
	if err != nil {
		t.Fatalf("ActiveRuns: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("active runs: got %d want 2", len(active))
	}

	stopResponse := postStop(t, stack.server, "agent-a", runA.RunID)
	if stopResponse.Status != string(domain.AgentRunStatusStopping) {
		t.Fatalf("stop status: got %q", stopResponse.Status)
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-a", runA.RunID, domain.AgentRunStatusStopped)

	recovery := appruntime.NewRecoveryUseCase(stack.agentRepo, stack.runtimeDBs)
	result, err := recovery.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(result.InterruptedRuns) != 1 {
		t.Fatalf("interrupted runs: got %d want 1", len(result.InterruptedRuns))
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-b", runB.RunID, domain.AgentRunStatusInterrupted)
}

type runtimeStack struct {
	server     *daemonhttp.Server
	manager    *infraruntime.Manager
	runtimeDBs *repository.RuntimeDBManager
	agentRepo  *repository.AgentRepository
}

func newRuntimeStack(t *testing.T) runtimeStack {
	t.Helper()

	return newRuntimeStackWithProvider(t, blockingE2EProvider{})
}

func newRuntimeStackWithProvider(t *testing.T, provider appruntime.Provider) runtimeStack {
	t.Helper()

	dir := t.TempDir()
	settingsDB, err := db.New("settings", db.Config{
		Path:         filepath.Join(dir, "settings.db"),
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
	agentRepo, err := repository.NewAgentRepository(settingsDB)
	if err != nil {
		t.Fatalf("NewAgentRepository: %v", err)
	}
	runtimeDBs, err := repository.NewRuntimeDBManager(filepath.Join(dir, "runtime"), 1)
	if err != nil {
		t.Fatalf("NewRuntimeDBManager: %v", err)
	}
	t.Cleanup(func() {
		if err := runtimeDBs.Close(context.Background()); err != nil {
			t.Fatalf("Close runtime DBs: %v", err)
		}
	})
	logFactory, err := runlogs.NewRunLogFactory(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("NewRunLogFactory: %v", err)
	}
	logReader, err := runlogs.NewRunLogReader(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("NewRunLogReader: %v", err)
	}
	isolation, err := infraruntime.NewIsolationBuilder(filepath.Join(dir, "work"))
	if err != nil {
		t.Fatalf("NewIsolationBuilder: %v", err)
	}
	manager, err := infraruntime.NewManager(
		runtimeDBs,
		logFactory,
		isolation,
		[]appruntime.Provider{provider},
	)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	applyUC, err := appagent.NewApplyUseCase(
		appagent.ParserFunc(definition.ParseMarkdown),
		agentRepo,
		runtimeDBs,
	)
	if err != nil {
		t.Fatalf("NewApplyUseCase: %v", err)
	}
	executeUC := appruntime.NewExecuteUseCase(agentRepo, manager)
	stopUC := appruntime.NewStopUseCase(manager)
	listUC, err := appagent.NewListUseCase(agentRepo)
	if err != nil {
		t.Fatalf("NewListUseCase: %v", err)
	}
	inspectUC, err := appagent.NewInspectUseCase(agentRepo)
	if err != nil {
		t.Fatalf("NewInspectUseCase: %v", err)
	}
	logsUC, err := applogs.NewUseCase(agentRepo, runtimeDBs, logReader)
	if err != nil {
		t.Fatalf("NewLogsUseCase: %v", err)
	}
	server := daemonhttp.NewServer(daemonhttp.Config{},
		daemonhttp.WithApplyUseCase(applyUC),
		daemonhttp.WithExecuteUseCase(executeUC),
		daemonhttp.WithStopUseCase(stopUC),
		daemonhttp.WithListUseCase(listUC),
		daemonhttp.WithInspectUseCase(inspectUC),
		daemonhttp.WithLogsUseCase(logsUC),
	)

	return runtimeStack{
		server:     server,
		manager:    manager,
		runtimeDBs: runtimeDBs,
		agentRepo:  agentRepo,
	}
}

type blockingE2EProvider struct{}

func (blockingE2EProvider) Name() string {
	return "openai"
}

func (blockingE2EProvider) Execute(
	ctx context.Context,
	_ appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return appruntime.ProviderResponse{}, ctx.Err()
	}

	return appruntime.ProviderResponse{}, nil
}

func postRun(t *testing.T, server *daemonhttp.Server, agentName string) model.RunResponse {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/v1/agents/"+agentName+"/runs", nil)
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

func postStop(
	t *testing.T,
	server *daemonhttp.Server,
	agentName string,
	runID string,
) model.RunResponse {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/agents/"+agentName+"/runs/"+runID+"/stop",
		nil,
	)
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("stop status: got %d body %s", response.Code, response.Body.String())
	}

	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode stop response: %v", err)
	}

	return body
}

func waitForE2ERunStatus(
	t *testing.T,
	runtimeDBs *repository.RuntimeDBManager,
	agentName string,
	runID string,
	want domain.AgentRunStatus,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := runtimeDBs.Runs(agentName).FindByID(context.Background(), runID)
		if err == nil && run.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, err := runtimeDBs.Runs(agentName).FindByID(context.Background(), runID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	t.Fatalf("run status: got %q want %q", run.Status, want)
}

func runtimeDefinition(name string) string {
	return `---
name: ` + name + `
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
Prompt for ` + name
}
