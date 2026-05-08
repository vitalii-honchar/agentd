package runtime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	runlogs "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/logs"
)

func BenchmarkFiveConcurrentAgentRunsSeparateRuntimeDBs(b *testing.B) {
	dir := b.TempDir()
	runtimeDBs, err := repository.NewRuntimeDBManager(filepath.Join(dir, "dbs"), 1)
	if err != nil {
		b.Fatalf("NewRuntimeDBManager: %v", err)
	}
	b.Cleanup(func() {
		if err := runtimeDBs.Close(context.Background()); err != nil {
			b.Fatalf("Close runtime DBs: %v", err)
		}
	})
	logFactory, err := runlogs.NewRunLogFactory(filepath.Join(dir, "logs"))
	if err != nil {
		b.Fatalf("NewRunLogFactory: %v", err)
	}
	isolation, err := NewIsolationBuilder(filepath.Join(dir, "work"))
	if err != nil {
		b.Fatalf("NewIsolationBuilder: %v", err)
	}
	manager, err := NewManager(runtimeDBs, logFactory, isolation, []appruntime.Provider{instantBenchmarkProvider{}})
	if err != nil {
		b.Fatalf("NewManager: %v", err)
	}
	agents := []domain.Agent{
		testAgent("bench-agent-a"),
		testAgent("bench-agent-b"),
		testAgent("bench-agent-c"),
		testAgent("bench-agent-d"),
		testAgent("bench-agent-e"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runs := make([]domain.AgentRun, 0, len(agents))
		for _, agent := range agents {
			run, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
				Agent:   agent,
				Trigger: domain.RunTriggerManual,
			})
			if err != nil {
				b.Fatalf("Execute %s: %v", agent.Name, err)
			}
			runs = append(runs, run)
		}
		for _, run := range runs {
			waitForBenchmarkRun(b, runtimeDBs, run)
		}
	}
}

type instantBenchmarkProvider struct{}

func (instantBenchmarkProvider) Name() string {
	return "openai"
}

func (instantBenchmarkProvider) Execute(
	context.Context,
	appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	return appruntime.ProviderResponse{Output: "ok"}, nil
}

func waitForBenchmarkRun(
	b *testing.B,
	runtimeDBs *repository.RuntimeDBManager,
	run domain.AgentRun,
) {
	b.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		found, err := runtimeDBs.Runs(run.AgentName).FindByID(context.Background(), run.ID)
		if err == nil && found.Status == domain.AgentRunStatusCompleted {
			return
		}
		time.Sleep(time.Millisecond)
	}
	b.Fatalf("run %s for %s did not complete", run.ID, run.AgentName)
}
