package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
)

func TestApplyUseCaseCreatedUpdatedUnchanged(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	created, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply created: %v", err)
	}
	if created.Outcome != ApplyOutcomeCreated {
		t.Fatalf("created outcome: got %q want %q", created.Outcome, ApplyOutcomeCreated)
	}
	if !runtimeDBs.ensured["release-notes-helper"] {
		t.Fatal("runtime DB was not ensured for created Agent")
	}

	unchanged, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply unchanged: %v", err)
	}
	if unchanged.Outcome != ApplyOutcomeUnchanged {
		t.Fatalf("unchanged outcome: got %q want %q", unchanged.Outcome, ApplyOutcomeUnchanged)
	}

	updated, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize updated changes."),
	})
	if err != nil {
		t.Fatalf("Apply updated: %v", err)
	}
	if updated.Outcome != ApplyOutcomeUpdated {
		t.Fatalf("updated outcome: got %q want %q", updated.Outcome, ApplyOutcomeUpdated)
	}
	if updated.Agent.Revision == created.Agent.Revision {
		t.Fatalf("updated revision should differ from created revision %q", created.Agent.Revision)
	}
}

func TestApplyUseCaseRejectsInvalidDefinitionWithoutMutation(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	_, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "bad.md",
		Markdown: `---
name: Bad Name
schedule:
  type: manual
vendor:
  name: openai
  model: ""
---
`,
	})
	if err == nil {
		t.Fatal("Apply returned nil error")
	}
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("Apply error %v does not match ErrInvalidDefinition", err)
	}
	if len(repo.agents) != 0 {
		t.Fatalf("repo mutated after invalid apply: %#v", repo.agents)
	}
	if len(runtimeDBs.ensured) != 0 {
		t.Fatalf("runtime DBs mutated after invalid apply: %#v", runtimeDBs.ensured)
	}
}

func newApplyUseCaseForTest(
	t *testing.T,
	repo *memoryAgentRepository,
	runtimeDBs *memoryRuntimeDBManager,
) *ApplyUseCase {
	t.Helper()

	useCase, err := NewApplyUseCase(
		ParserFunc(definition.ParseMarkdown),
		repo,
		runtimeDBs,
	)
	if err != nil {
		t.Fatalf("NewApplyUseCase: %v", err)
	}

	return useCase
}

type memoryAgentRepository struct {
	agents map[string]domain.Agent
}

func newMemoryAgentRepository() *memoryAgentRepository {
	return &memoryAgentRepository{agents: make(map[string]domain.Agent)}
}

func (r *memoryAgentRepository) Save(
	_ context.Context,
	agent domain.Agent,
	_ []domain.ToolPermission,
	_ []domain.ToolPermission,
) error {
	r.agents[agent.Name] = agent

	return nil
}

func (r *memoryAgentRepository) FindByName(_ context.Context, name string) (domain.Agent, error) {
	agent, ok := r.agents[name]
	if !ok {
		return domain.Agent{}, domain.ErrNotFound
	}

	return agent, nil
}

func (r *memoryAgentRepository) List(context.Context) ([]domain.Agent, error) {
	agents := make([]domain.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

type memoryRuntimeDBManager struct {
	ensured map[string]bool
}

func (m *memoryRuntimeDBManager) EnsureAgent(_ context.Context, agentName string) error {
	if m.ensured == nil {
		m.ensured = make(map[string]bool)
	}
	m.ensured[agentName] = true

	return nil
}

func (m *memoryRuntimeDBManager) Runs(string) app.AgentRunRepository {
	return nil
}

func (m *memoryRuntimeDBManager) Events(string) app.RuntimeEventRepository {
	return nil
}

func (m *memoryRuntimeDBManager) Close(context.Context) error {
	return nil
}

func manualDefinition(prompt string) string {
	return `---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.openai.com"]
---
` + prompt
}
