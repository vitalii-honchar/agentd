package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	runlogs "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/logs"
)

func TestManagerRunsDifferentAgentsConcurrentlyAndIsolatesLogs(t *testing.T) {
	t.Parallel()

	manager, _ := newManagerFixture(t, &blockingProvider{name: "openai"})
	agentA := testAgent("agent-a")
	agentB := testAgent("agent-b")

	runA, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agentA, Trigger: domain.RunTriggerManual,
	})
	if err != nil {
		t.Fatalf("Execute A: %v", err)
	}
	runB, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agentB, Trigger: domain.RunTriggerManual,
	})
	if err != nil {
		t.Fatalf("Execute B: %v", err)
	}
	if runA.WorkDir == runB.WorkDir {
		t.Fatalf("work dirs are not isolated: %q", runA.WorkDir)
	}
	if runA.LogPath == runB.LogPath {
		t.Fatalf("log paths are not isolated: %q", runA.LogPath)
	}

	active, err := manager.ActiveRuns(context.Background())
	if err != nil {
		t.Fatalf("ActiveRuns: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("active runs: got %d want 2", len(active))
	}
}

func TestManagerRejectsSameAgentOverlap(t *testing.T) {
	t.Parallel()

	manager, _ := newManagerFixture(t, &blockingProvider{name: "openai"})
	agent := testAgent("agent-a")
	if _, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agent, Trigger: domain.RunTriggerManual,
	}); err != nil {
		t.Fatalf("Execute first: %v", err)
	}
	_, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agent, Trigger: domain.RunTriggerManual,
	})
	if !errors.Is(err, domain.ErrRunAlreadyActive) {
		t.Fatalf("Execute second error: got %v want %v", err, domain.ErrRunAlreadyActive)
	}
}

func TestManagerStopCancelsRun(t *testing.T) {
	t.Parallel()

	provider := &blockingProvider{name: "openai"}
	manager, runtimeDBs := newManagerFixture(t, provider)
	agent := testAgent("agent-a")
	run, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agent, Trigger: domain.RunTriggerManual,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	stopping, err := manager.Stop(context.Background(), appruntime.StopRequest{
		AgentName: agent.Name,
		RunID:     run.ID,
	})
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopping.Status != domain.AgentRunStatusStopping {
		t.Fatalf("stop status: got %q", stopping.Status)
	}

	waitForRunStatus(t, runtimeDBs, agent.Name, run.ID, domain.AgentRunStatusStopped)
}

func TestManagerDoesNotExecuteUndeclaredTools(t *testing.T) {
	t.Parallel()

	provider := &capturingProvider{name: "openai", output: "analysis complete"}
	manager, runtimeDBs := newManagerFixture(t, provider)
	manager.SetToolExecutor(&recordingToolExecutor{
		t:             t,
		failOnExecute: true,
	})
	agent := testAgent("manual-agent")
	agent.Prompt = "Run tools/undeclared.sh before answering."

	run, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agent, Trigger: domain.RunTriggerManual,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	waitForRunStatus(t, runtimeDBs, agent.Name, run.ID, domain.AgentRunStatusCompleted)
	if strings.Contains(provider.prompt(), "Tool results:") {
		t.Fatalf("provider prompt included undeclared tool output: %q", provider.prompt())
	}
}

func TestManagerExecutesDeclaredLocalToolsBeforeProvider(t *testing.T) {
	t.Parallel()

	provider := &capturingProvider{name: "openai", output: "analysis complete"}
	manager, runtimeDBs := newManagerFixture(t, provider)
	toolExecutor := &recordingToolExecutor{
		result: appruntime.ToolResult{StdoutSummary: "snapshot title: Example"},
	}
	manager.SetToolExecutor(toolExecutor)
	agent := testAgent("website-snapshot-analyst")
	agent.DefinitionSource = filepath.Join(t.TempDir(), "website-snapshot-analyst.md")
	agent.Tools = []domain.ToolPermission{{
		Name:    "snapshot",
		Kind:    domain.ToolKindLocalTool,
		Command: "tools/snapshot.js",
		Args:    []string{"--url", "{{inputs.url}}"},
	}}

	run, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent: agent, Trigger: domain.RunTriggerManual, Inputs: map[string]string{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	waitForRunStatus(t, runtimeDBs, agent.Name, run.ID, domain.AgentRunStatusCompleted)
	if !strings.Contains(provider.prompt(), "Tool results:\nsnapshot stdout: snapshot title: Example") {
		t.Fatalf("provider prompt missing tool output: %q", provider.prompt())
	}
	if got := toolExecutor.request().Tool.Args[1]; got != "https://example.com" {
		t.Fatalf("tool input arg: got %q", got)
	}
	events, err := runtimeDBs.Events(agent.Name).ListByRun(context.Background(), run.ID, 20)
	if err != nil {
		t.Fatalf("ListByRun: %v", err)
	}
	assertEventType(t, events, domain.RunActionToolExecuteStart)
	assertEventType(t, events, domain.RunActionToolExecuteComplete)
}

func newManagerFixture(t *testing.T, provider appruntime.Provider) (*Manager, app.RuntimeDBManager) {
	t.Helper()

	dir := t.TempDir()
	runtimeDBs, err := repository.NewRuntimeDBManager(filepath.Join(dir, "dbs"), 1)
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
	isolation, err := NewIsolationBuilder(filepath.Join(dir, "work"))
	if err != nil {
		t.Fatalf("NewIsolationBuilder: %v", err)
	}
	manager, err := NewManager(runtimeDBs, logFactory, isolation, []appruntime.Provider{provider})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	return manager, runtimeDBs
}

type blockingProvider struct {
	name string
}

func (p *blockingProvider) Name() string {
	return p.name
}

func (p *blockingProvider) Execute(ctx context.Context, _ appruntime.ProviderRequest) (appruntime.ProviderResponse, error) {
	<-ctx.Done()

	return appruntime.ProviderResponse{}, ctx.Err()
}

type capturingProvider struct {
	name   string
	output string

	mu         sync.Mutex
	lastPrompt string
}

func (p *capturingProvider) Name() string {
	return p.name
}

func (p *capturingProvider) Execute(_ context.Context, request appruntime.ProviderRequest) (appruntime.ProviderResponse, error) {
	p.mu.Lock()
	p.lastPrompt = request.Prompt
	p.mu.Unlock()

	return appruntime.ProviderResponse{RequestID: "request-1", Output: p.output}, nil
}

func (p *capturingProvider) prompt() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.lastPrompt
}

type recordingToolExecutor struct {
	t             *testing.T
	failOnExecute bool
	result        appruntime.ToolResult

	mu          sync.Mutex
	lastRequest appruntime.ToolRequest
}

func (e *recordingToolExecutor) Execute(_ context.Context, request appruntime.ToolRequest) (appruntime.ToolResult, error) {
	e.mu.Lock()
	e.lastRequest = request
	e.mu.Unlock()
	if e.failOnExecute {
		e.t.Fatal("undeclared tool was executed")
	}

	return e.result, nil
}

func (e *recordingToolExecutor) request() appruntime.ToolRequest {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.lastRequest
}

func testAgent(name string) domain.Agent {
	return domain.Agent{
		Name:     name,
		Revision: "rev-1",
		Enabled:  true,
		Status:   domain.AgentStatusActive,
		Vendor:   domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule: domain.Schedule{Type: domain.ScheduleTypeManual},
		Prompt:   "prompt",
	}
}

func waitForRunStatus(
	t *testing.T,
	runtimeDBs app.RuntimeDBManager,
	agentName string,
	runID string,
	want domain.AgentRunStatus,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := runtimeDBs.Runs(agentName).FindByID(context.Background(), runID)
		if err == nil && run.Status == want {
			if _, err := os.Stat(run.LogPath); err != nil {
				t.Fatalf("run log does not exist: %v", err)
			}

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

func assertEventType(t *testing.T, events []domain.RuntimeEvent, eventType string) {
	t.Helper()

	for _, event := range events {
		if event.EventType == eventType {
			return
		}
	}
	t.Fatalf("event %q not found in %#v", eventType, events)
}

func TestIsolationBuilderCreatesPerRunDirectory(t *testing.T) {
	t.Parallel()

	builder, err := NewIsolationBuilder(t.TempDir())
	if err != nil {
		t.Fatalf("NewIsolationBuilder: %v", err)
	}
	env, err := builder.Build(testAgent("agent-a"), "run-1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(env.WorkDir); err != nil {
		t.Fatalf("work dir was not created: %v", err)
	}
}

func TestRunLogFactoryCreatesIsolatedLog(t *testing.T) {
	t.Parallel()

	factory, err := runlogs.NewRunLogFactory(t.TempDir())
	if err != nil {
		t.Fatalf("NewRunLogFactory: %v", err)
	}
	writer, err := factory.Create(context.Background(), "agent-a", "run-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := writer.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	body, err := os.ReadFile(writer.Path())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("log body: got %q", body)
	}
}
