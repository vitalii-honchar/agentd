package runtime

import (
	"context"
	"path/filepath"
	"testing"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestIsolationBuilderUsesPortableNestedPath(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	builder, err := NewIsolationBuilder(baseDir)
	if err != nil {
		t.Fatalf("NewIsolationBuilder: %v", err)
	}

	env, err := builder.Build(testAgent("agent-a"), "run-1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	relative, err := filepath.Rel(baseDir, env.WorkDir)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	if relative != filepath.Join("agent-a", "run-1") {
		t.Fatalf("relative path: got %q want %q", relative, filepath.Join("agent-a", "run-1"))
	}
}

func TestStopCancellationLeavesNoActiveRun(t *testing.T) {
	t.Parallel()

	manager, runtimeDBs := newManagerFixture(t, &blockingProvider{name: "openai"})
	agent := testAgent("agent-a")
	run, err := manager.Execute(context.Background(), appruntime.ExecuteRequest{
		Agent:   agent,
		Trigger: domain.RunTriggerManual,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, err := manager.Stop(context.Background(), appruntime.StopRequest{
		AgentName: agent.Name,
		RunID:     run.ID,
	}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitForRunStatus(t, runtimeDBs, agent.Name, run.ID, domain.AgentRunStatusStopped)

	active, err := manager.ActiveRuns(context.Background())
	if err != nil {
		t.Fatalf("ActiveRuns: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active runs after stop: %#v", active)
	}
}
