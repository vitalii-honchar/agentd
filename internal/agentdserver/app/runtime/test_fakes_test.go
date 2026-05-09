package runtime

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type fakeManager struct {
	executeRequest ExecuteRequest
	executeCalled  bool
	stopRequest    StopRequest
	run            domain.AgentRun
	err            error
}

func (m *fakeManager) Execute(_ context.Context, request ExecuteRequest) (domain.AgentRun, error) {
	m.executeCalled = true
	m.executeRequest = request
	if m.err != nil {
		return domain.AgentRun{}, m.err
	}

	return m.run, nil
}

func (m *fakeManager) Stop(_ context.Context, request StopRequest) (domain.AgentRun, error) {
	m.stopRequest = request
	if m.err != nil {
		return domain.AgentRun{}, m.err
	}

	return m.run, nil
}

func (m *fakeManager) Recover(context.Context) (RecoveryResult, error) {
	if m.err != nil {
		return RecoveryResult{}, m.err
	}

	return RecoveryResult{InterruptedRuns: []domain.AgentRun{m.run}}, nil
}

func (m *fakeManager) ActiveRuns(context.Context) ([]domain.AgentRun, error) {
	if m.err != nil {
		return nil, m.err
	}

	return []domain.AgentRun{m.run}, nil
}

type runtimeAgentRepo struct {
	agents    map[string]domain.Agent
	revisions []domain.AgentRevision
}

func newRuntimeAgentRepo(agents ...domain.Agent) *runtimeAgentRepo {
	repo := &runtimeAgentRepo{agents: make(map[string]domain.Agent)}
	for _, agent := range agents {
		repo.agents[agent.Name] = agent
	}

	return repo
}

func (r *runtimeAgentRepo) Save(
	context.Context,
	domain.Agent,
	[]domain.ToolPermission,
	[]domain.ToolPermission,
) error {
	return nil
}

func (r *runtimeAgentRepo) FindByName(_ context.Context, name string) (domain.Agent, error) {
	agent, ok := r.agents[name]
	if !ok {
		return domain.Agent{}, domain.ErrNotFound
	}

	return agent, nil
}

func (r *runtimeAgentRepo) List(context.Context) ([]domain.Agent, error) {
	agents := make([]domain.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

func (r *runtimeAgentRepo) SaveRevision(context.Context, domain.AgentRevision) error {
	return nil
}

func (r *runtimeAgentRepo) ListRevisions(_ context.Context, agentName string) ([]domain.AgentRevision, error) {
	var revisions []domain.AgentRevision
	for _, revision := range r.revisions {
		if revision.AgentName == agentName {
			revisions = append(revisions, revision)
		}
	}

	return revisions, nil
}

func (r *runtimeAgentRepo) FindRevisionByID(
	_ context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, error) {
	for _, revision := range r.revisions {
		if revision.AgentName == agentName && revision.RevisionID == revisionID {
			return revision, nil
		}
	}

	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *runtimeAgentRepo) FindRevisionByDigest(context.Context, string, string) (domain.AgentRevision, error) {
	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *runtimeAgentRepo) FindLatestFinalizedRevision(
	_ context.Context,
	agentName string,
) (domain.AgentRevision, error) {
	for i := len(r.revisions) - 1; i >= 0; i-- {
		revision := r.revisions[i]
		if revision.AgentName == agentName && revision.Status == domain.AgentRevisionStatusFinalized {
			return revision, nil
		}
	}

	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *runtimeAgentRepo) MarkRevisionCorrupt(context.Context, string, string, string) error {
	return nil
}

type memoryRuntimeDBs struct {
	runs map[string]app.AgentRunRepository
}

func (m *memoryRuntimeDBs) EnsureAgent(context.Context, string) error {
	return nil
}

func (m *memoryRuntimeDBs) Runs(agentName string) app.AgentRunRepository {
	return m.runs[agentName]
}

func (m *memoryRuntimeDBs) Events(string) app.RuntimeEventRepository {
	return nil
}

func (m *memoryRuntimeDBs) Close(context.Context) error {
	return nil
}

type memoryRunRepo struct {
	active  []domain.AgentRun
	updated []domain.AgentRun
}

func (r *memoryRunRepo) Create(context.Context, domain.AgentRun) error {
	return nil
}

func (r *memoryRunRepo) Update(_ context.Context, run domain.AgentRun) error {
	r.updated = append(r.updated, run)

	return nil
}

func (r *memoryRunRepo) FindByID(context.Context, string) (domain.AgentRun, error) {
	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *memoryRunRepo) FindLatest(context.Context) (domain.AgentRun, error) {
	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *memoryRunRepo) FindActive(context.Context) (domain.AgentRun, error) {
	if len(r.active) == 0 {
		return domain.AgentRun{}, domain.ErrNotFound
	}

	return r.active[0], nil
}

func (r *memoryRunRepo) List(context.Context) ([]domain.AgentRun, error) {
	return r.active, nil
}

func (r *memoryRunRepo) ListActive(context.Context) ([]domain.AgentRun, error) {
	return r.active, nil
}

func (r *memoryRunRepo) ListTerminal(context.Context) ([]domain.AgentRun, error) {
	return nil, nil
}

func (r *memoryRunRepo) CreateToolExecution(context.Context, domain.ToolExecution) error {
	return nil
}

func testRuntimeAgent(name string) domain.Agent {
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

func testRuntimeRevision(agentName, revisionID, prompt string) domain.AgentRevision {
	return domain.AgentRevision{
		AgentName:    agentName,
		RevisionID:   revisionID,
		Prompt:       prompt,
		Vendor:       domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:     domain.Schedule{Type: domain.ScheduleTypeManual},
		Status:       domain.AgentRevisionStatusFinalized,
		ArtifactPath: "data/work/" + agentName + "/" + revisionID,
		Tools: []domain.RevisionTool{{
			Name:             "fetch",
			Kind:             domain.ToolKindCustomTool,
			OriginalCommand:  "tools/fetch.py",
			RewrittenCommand: "artifact/tools/fetch.py",
		}},
	}
}
