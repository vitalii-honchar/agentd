package agent

import (
	"context"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestInspectUseCaseFindsAgentByName(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	repo.agents["release-notes-helper"] = testAgent("release-notes-helper")
	useCase, err := NewInspectUseCase(repo)
	if err != nil {
		t.Fatalf("NewInspectUseCase: %v", err)
	}

	agent, err := useCase.Inspect(context.Background(), "release-notes-helper")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if agent.Name != "release-notes-helper" {
		t.Fatalf("agent name: got %q", agent.Name)
	}
}

func TestInspectUseCaseReturnsNotFound(t *testing.T) {
	t.Parallel()

	useCase, err := NewInspectUseCase(newMemoryAgentRepository())
	if err != nil {
		t.Fatalf("NewInspectUseCase: %v", err)
	}

	_, err = useCase.Inspect(context.Background(), "missing")
	if err != domain.ErrNotFound {
		t.Fatalf("Inspect error: got %v want %v", err, domain.ErrNotFound)
	}
}

func TestListUseCaseReturnsAgents(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	repo.agents["daily-pr-review"] = testAgent("daily-pr-review")
	repo.agents["release-notes-helper"] = testAgent("release-notes-helper")
	useCase, err := NewListUseCase(repo)
	if err != nil {
		t.Fatalf("NewListUseCase: %v", err)
	}

	agents, err := useCase.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("agents length: got %d want 2", len(agents))
	}
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
