package runtime

import (
	"context"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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

func TestRecoveryUseCaseMarksActiveReActProviderRunsTerminal(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("react-agent")
	repo := newRuntimeAgentRepo(agent)
	runs := &memoryRunRepo{active: []domain.AgentRun{{
		ID:            "run-react",
		AgentName:     agent.Name,
		Status:        domain.AgentRunStatusRunning,
		ProviderName:  "codex",
		ProviderModel: "gpt-5",
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
	updated := runs.updated[0]
	if updated.Status != domain.AgentRunStatusInterrupted || !updated.IsTerminal() {
		t.Fatalf("updated status: %#v", updated)
	}
	if updated.ErrorCode != "run_interrupted" {
		t.Fatalf("error code: got %q", updated.ErrorCode)
	}
	if updated.Result == "" || updated.ResultSummary == "" {
		t.Fatalf("interrupted run should have result text: %#v", updated)
	}
}
