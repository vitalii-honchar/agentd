package runtime

import (
	"context"
	"errors"
	"testing"

	"agentd/internal/agentdserver/domain"
)

func TestExecuteUseCaseStartsManualRun(t *testing.T) {
	t.Parallel()

	repo := newRuntimeAgentRepo(testRuntimeAgent("release-notes-helper"))
	manager := &fakeManager{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		Status:    domain.AgentRunStatusRunning,
	}}
	useCase := NewExecuteUseCase(repo, manager)

	run, err := useCase.Execute(context.Background(), "release-notes-helper")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.ID != "run-1" {
		t.Fatalf("run id: got %q", run.ID)
	}
	if manager.executeRequest.Trigger != domain.RunTriggerManual {
		t.Fatalf("trigger: got %q", manager.executeRequest.Trigger)
	}
}

func TestExecuteUseCaseRejectsDisabledAgent(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("release-notes-helper")
	agent.Enabled = false
	repo := newRuntimeAgentRepo(agent)
	useCase := NewExecuteUseCase(repo, &fakeManager{})

	_, err := useCase.Execute(context.Background(), agent.Name)
	if !errors.Is(err, domain.ErrAgentDisabled) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrAgentDisabled)
	}
}

func TestExecuteUseCaseUnknownAgent(t *testing.T) {
	t.Parallel()

	useCase := NewExecuteUseCase(newRuntimeAgentRepo(), &fakeManager{})
	_, err := useCase.Execute(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrNotFound)
	}
}

func TestExecuteUseCasePropagatesSameAgentOverlap(t *testing.T) {
	t.Parallel()

	repo := newRuntimeAgentRepo(testRuntimeAgent("release-notes-helper"))
	useCase := NewExecuteUseCase(repo, &fakeManager{err: domain.ErrRunAlreadyActive})
	_, err := useCase.Execute(context.Background(), "release-notes-helper")
	if !errors.Is(err, domain.ErrRunAlreadyActive) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrRunAlreadyActive)
	}
}
