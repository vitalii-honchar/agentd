package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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

	run, err := useCase.Execute(context.Background(), "release-notes-helper", nil)
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

func TestExecuteUseCasePassesDeclaredToolsToManager(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("website-snapshot-analyst")
	agent.Tools = []domain.ToolPermission{{
		Name:    "snapshot",
		Kind:    domain.ToolKindLocalTool,
		Command: "tools/snapshot.js",
	}}
	repo := newRuntimeAgentRepo(agent)
	manager := &fakeManager{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: agent.Name,
		Status:    domain.AgentRunStatusRunning,
	}}
	useCase := NewExecuteUseCase(repo, manager)

	if _, err := useCase.Execute(context.Background(), agent.Name, map[string]string{"url": "https://example.com"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(manager.executeRequest.Agent.Tools) != 1 {
		t.Fatalf("tools: got %#v", manager.executeRequest.Agent.Tools)
	}
	if manager.executeRequest.Agent.Tools[0].Command != "tools/snapshot.js" {
		t.Fatalf("tool command: got %q", manager.executeRequest.Agent.Tools[0].Command)
	}
	if manager.executeRequest.Inputs["url"] != "https://example.com" {
		t.Fatalf("inputs: %#v", manager.executeRequest.Inputs)
	}
}

func TestExecuteUseCaseResolvesLatestFinalizedRevision(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("release-notes-helper")
	agent.Revision = "latest-rev"
	repo := newRuntimeAgentRepo(agent)
	repo.revisions = []domain.AgentRevision{
		testRuntimeRevision(agent.Name, "older-rev", "Older prompt"),
		testRuntimeRevision(agent.Name, "latest-rev", "Frozen latest prompt"),
	}
	manager := &fakeManager{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: agent.Name,
		Status:    domain.AgentRunStatusRunning,
	}}
	useCase := NewExecuteUseCase(repo, manager)

	if _, err := useCase.Execute(context.Background(), agent.Name, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if manager.executeRequest.Agent.Revision != "latest-rev" {
		t.Fatalf("revision: got %q", manager.executeRequest.Agent.Revision)
	}
	if manager.executeRequest.Agent.Prompt != "Frozen latest prompt" {
		t.Fatalf("prompt: got %q", manager.executeRequest.Agent.Prompt)
	}
	if len(manager.executeRequest.Agent.Tools) != 1 || manager.executeRequest.Agent.Tools[0].Command != "artifact/tools/fetch.py" {
		t.Fatalf("tools: %#v", manager.executeRequest.Agent.Tools)
	}
}

func TestExecuteUseCaseResolvesExplicitRevision(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("release-notes-helper")
	agent.Revision = "latest-rev"
	repo := newRuntimeAgentRepo(agent)
	repo.revisions = []domain.AgentRevision{
		testRuntimeRevision(agent.Name, "older-rev", "Older prompt"),
		testRuntimeRevision(agent.Name, "latest-rev", "Frozen latest prompt"),
	}
	manager := &fakeManager{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: agent.Name,
		Status:    domain.AgentRunStatusRunning,
	}}
	useCase := NewExecuteUseCase(repo, manager)

	if _, err := useCase.Execute(context.Background(), agent.Name+":older-rev", nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if manager.executeRequest.Agent.Revision != "older-rev" {
		t.Fatalf("revision: got %q", manager.executeRequest.Agent.Revision)
	}
	if manager.executeRequest.Agent.Prompt != "Older prompt" {
		t.Fatalf("prompt: got %q", manager.executeRequest.Agent.Prompt)
	}
}

func TestExecuteUseCaseRejectsDisabledAgent(t *testing.T) {
	t.Parallel()

	agent := testRuntimeAgent("release-notes-helper")
	agent.Enabled = false
	repo := newRuntimeAgentRepo(agent)
	useCase := NewExecuteUseCase(repo, &fakeManager{})

	_, err := useCase.Execute(context.Background(), agent.Name, nil)
	if !errors.Is(err, domain.ErrAgentDisabled) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrAgentDisabled)
	}
}

func TestExecuteUseCaseUnknownAgent(t *testing.T) {
	t.Parallel()

	useCase := NewExecuteUseCase(newRuntimeAgentRepo(), &fakeManager{})
	_, err := useCase.Execute(context.Background(), "missing", nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrNotFound)
	}
}

func TestExecuteUseCasePropagatesSameAgentOverlap(t *testing.T) {
	t.Parallel()

	repo := newRuntimeAgentRepo(testRuntimeAgent("release-notes-helper"))
	useCase := NewExecuteUseCase(repo, &fakeManager{err: domain.ErrRunAlreadyActive})
	_, err := useCase.Execute(context.Background(), "release-notes-helper", nil)
	if !errors.Is(err, domain.ErrRunAlreadyActive) {
		t.Fatalf("Execute error: got %v want %v", err, domain.ErrRunAlreadyActive)
	}
}
