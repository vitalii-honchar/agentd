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

func TestRevisionUseCaseListsRevisionsWithLatestMarker(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	repo.agents["release-notes-helper"] = testAgent("release-notes-helper")
	repo.revisions = []domain.AgentRevision{
		{
			AgentName:         "release-notes-helper",
			RevisionID:        "older-rev",
			Status:            domain.AgentRevisionStatusFinalized,
			SourcePath:        "agent.md",
			ArtifactPath:      "data/work/release-notes-helper/older-rev",
			IsLatestFinalized: false,
		},
		{
			AgentName:         "release-notes-helper",
			RevisionID:        "latest-rev",
			Status:            domain.AgentRevisionStatusFinalized,
			SourcePath:        "agent.md",
			ArtifactPath:      "data/work/release-notes-helper/latest-rev",
			IsLatestFinalized: true,
		},
	}
	useCase, err := NewRevisionUseCase(repo)
	if err != nil {
		t.Fatalf("NewRevisionUseCase: %v", err)
	}

	revisions, err := useCase.ListRevisions(context.Background(), "release-notes-helper")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("revisions length: got %d want 2", len(revisions))
	}
	if revisions[1].RevisionID != "latest-rev" || !revisions[1].IsLatestFinalized {
		t.Fatalf("latest revision marker: %#v", revisions)
	}
}

func TestRevisionUseCaseInspectsRevisionDetailsWithMaskedEnvironment(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	repo.agents["release-notes-helper"] = testAgent("release-notes-helper")
	repo.revisions = []domain.AgentRevision{{
		AgentName:    "release-notes-helper",
		RevisionID:   "revision-1",
		Status:       domain.AgentRevisionStatusFinalized,
		ArtifactPath: "data/work/release-notes-helper/revision-1",
		Tools: []domain.RevisionTool{
			{
				Name:             "fetch",
				Kind:             domain.ToolKindCustomTool,
				OriginalCommand:  "tools/fetch.py",
				RewrittenCommand: "data/work/release-notes-helper/revision-1/tools/fetch.py",
				CopiedFiles:      []string{"tools/fetch.py"},
			},
			{
				Name:        "github_api",
				Kind:        domain.ToolKindHostTool,
				HostCommand: "gh",
			},
		},
		Environment: []domain.RevisionEnvironment{{
			Key:    "GITHUB_TOKEN",
			Value:  "secret",
			Source: domain.RevisionEnvironmentSourceLiteral,
			Masked: true,
		}},
	}}
	useCase, err := NewRevisionUseCase(repo)
	if err != nil {
		t.Fatalf("NewRevisionUseCase: %v", err)
	}

	revision, err := useCase.InspectRevision(context.Background(), "release-notes-helper", "revision-1")
	if err != nil {
		t.Fatalf("InspectRevision: %v", err)
	}
	if len(revision.Tools) != 2 {
		t.Fatalf("tools: %#v", revision.Tools)
	}
	if revision.Tools[0].Kind != domain.ToolKindCustomTool || revision.Tools[0].RewrittenCommand == "" || len(revision.Tools[0].CopiedFiles) != 1 {
		t.Fatalf("custom tool: %#v", revision.Tools[0])
	}
	if revision.Tools[1].Kind != domain.ToolKindHostTool || revision.Tools[1].HostCommand != "gh" {
		t.Fatalf("host tool: %#v", revision.Tools[1])
	}
	if len(revision.Environment) != 1 || revision.Environment[0].Value != "********" {
		t.Fatalf("masked environment: %#v", revision.Environment)
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
