package runtime

import (
	"context"
	"testing"

	"agentd/internal/agentdserver/app"
	"agentd/internal/agentdserver/domain"
)

func TestRecoveryUseCaseInterruptsActiveRuns(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("release-notes-helper")
	repo := newRuntimeAgentRepo(agent)
	runs := &memoryRunRepo{active: []domain.AgentRun{{
		ID:        "run-1",
		AgentName: agent.Name,
		Status:    domain.AgentRunStatusRunning,
	}}}
	runtimeDBs := &memoryRuntimeDBs{runs: map[string]app.AgentRunRepository{agent.Name: runs}}
	useCase := NewRecoveryUseCase(repo, runtimeDBs)

	result, err := useCase.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(result.InterruptedRuns) != 1 {
		t.Fatalf("interrupted runs: got %d", len(result.InterruptedRuns))
	}
	if runs.updated[0].Status != domain.AgentRunStatusInterrupted {
		t.Fatalf("updated status: got %q", runs.updated[0].Status)
	}
	if runs.updated[0].CompletedAt == nil {
		t.Fatal("completed_at was not set")
	}
}
