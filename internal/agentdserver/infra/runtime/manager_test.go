package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
