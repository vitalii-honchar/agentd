package logs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestUseCaseReadsLatestRunLogs(t *testing.T) {
	t.Parallel()

	run := domain.AgentRun{
		ID:        "run-latest",
		AgentName: "release-notes-helper",
		LogPath:   "/tmp/run-latest.log",
		Status:    domain.AgentRunStatusCompleted,
	}
	reader := &fakeLogReader{entries: []app.LogEntry{{RunID: run.ID, Line: "done"}}}
	useCase := newUseCaseForTest(t, reader, run)

	result, err := useCase.Read(context.Background(), Query{AgentName: "release-notes-helper", Tail: 10})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Run.ID != "run-latest" || len(result.Entries) != 1 {
		t.Fatalf("result: %#v", result)
	}
	if reader.query.RunID != "run-latest" || reader.query.LogPath != "/tmp/run-latest.log" || reader.query.Tail != 10 {
		t.Fatalf("reader query: %#v", reader.query)
	}
}

func TestUseCaseReadsSpecificRunLogs(t *testing.T) {
	t.Parallel()

	run := domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		LogPath:   "/tmp/run-1.log",
		Status:    domain.AgentRunStatusCompleted,
	}
	reader := &fakeLogReader{entries: []app.LogEntry{{RunID: run.ID, Line: "specific"}}}
	useCase := newUseCaseForTest(t, reader, run)

	result, err := useCase.Read(context.Background(), Query{
		AgentName: "release-notes-helper",
		RunID:     "run-1",
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Run.ID != "run-1" || result.Entries[0].Line != "specific" {
		t.Fatalf("result: %#v", result)
	}
}

func TestUseCaseIncludesScopedRuntimeActionLogs(t *testing.T) {
	t.Parallel()

	run := domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		LogPath:   "/tmp/run-1.log",
		Status:    domain.AgentRunStatusCompleted,
	}
	event := domain.RuntimeEvent{
		ID:        "event-1",
		AgentName: "release-notes-helper",
		RunID:     "run-1",
		EventType: domain.RunActionLLMPromptSend,
		Level:     domain.EventLevelInfo,
		Message:   "sent prompt to provider",
		CreatedAt: time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC),
	}
	useCase, err := NewUseCase(
		newMemoryAgentRepository(testAgent("release-notes-helper")),
		&memoryRuntimeDBManager{
			runs: map[string]app.AgentRunRepository{
				"release-notes-helper": &memoryRunRepository{runs: []domain.AgentRun{run}},
			},
			events: map[string]app.RuntimeEventRepository{
				"release-notes-helper": &memoryEventRepository{events: []domain.RuntimeEvent{event}},
			},
		},
		&fakeLogReader{},
	)
	if err != nil {
		t.Fatalf("NewUseCase: %v", err)
	}

	result, err := useCase.Read(context.Background(), Query{AgentName: "release-notes-helper", RunID: "run-1"})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Action != domain.RunActionLLMPromptSend {
		t.Fatalf("entries: %#v", result.Entries)
	}
	if result.Entries[0].Message != "sent prompt to provider" {
		t.Fatalf("message: %#v", result.Entries[0])
	}
}

func TestUseCaseReturnsEmptyLogsForEmptyFile(t *testing.T) {
	t.Parallel()

	run := domain.AgentRun{
		ID:        "run-empty",
		AgentName: "release-notes-helper",
		LogPath:   "/tmp/run-empty.log",
		Status:    domain.AgentRunStatusCompleted,
	}
	useCase := newUseCaseForTest(t, &fakeLogReader{}, run)

	result, err := useCase.Read(context.Background(), Query{AgentName: "release-notes-helper"})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("entries: %#v", result.Entries)
	}
}

func TestUseCaseReturnsNotFoundForPrunedLogs(t *testing.T) {
	t.Parallel()

	run := domain.AgentRun{
		ID:        "run-pruned",
		AgentName: "release-notes-helper",
		LogPath:   "/tmp/run-pruned.log",
		Status:    domain.AgentRunStatusCompleted,
	}
	useCase := newUseCaseForTest(t, &fakeLogReader{err: domain.ErrNotFound}, run)

	_, err := useCase.Read(context.Background(), Query{AgentName: "release-notes-helper"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Read error: got %v want ErrNotFound", err)
	}
}

func newUseCaseForTest(t *testing.T, reader app.RunLogReader, runs ...domain.AgentRun) *UseCase {
	t.Helper()

	useCase, err := NewUseCase(
		newMemoryAgentRepository(testAgent("release-notes-helper")),
		&memoryRuntimeDBManager{runs: map[string]app.AgentRunRepository{
			"release-notes-helper": &memoryRunRepository{runs: runs},
		}},
		reader,
	)
	if err != nil {
		t.Fatalf("NewUseCase: %v", err)
	}

	return useCase
}

type fakeLogReader struct {
	query   app.LogQuery
	entries []app.LogEntry
	err     error
}

func (r *fakeLogReader) Read(_ context.Context, query app.LogQuery) ([]app.LogEntry, error) {
	r.query = query
	if r.err != nil {
		return nil, r.err
	}

	return r.entries, nil
}

type memoryAgentRepository struct {
	agents map[string]domain.Agent
}

func newMemoryAgentRepository(agents ...domain.Agent) *memoryAgentRepository {
	repo := &memoryAgentRepository{agents: make(map[string]domain.Agent)}
	for _, agent := range agents {
		repo.agents[agent.Name] = agent
	}

	return repo
}

func (r *memoryAgentRepository) Save(context.Context, domain.Agent, []domain.ToolPermission, []domain.ToolPermission) error {
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
	runs   map[string]app.AgentRunRepository
	events map[string]app.RuntimeEventRepository
}

func (m *memoryRuntimeDBManager) EnsureAgent(context.Context, string) error {
	return nil
}

func (m *memoryRuntimeDBManager) Runs(agentName string) app.AgentRunRepository {
	return m.runs[agentName]
}

func (m *memoryRuntimeDBManager) Events(agentName string) app.RuntimeEventRepository {
	return m.events[agentName]
}

func (m *memoryRuntimeDBManager) Close(context.Context) error {
	return nil
}

type memoryRunRepository struct {
	runs []domain.AgentRun
}

func (r *memoryRunRepository) Create(context.Context, domain.AgentRun) error {
	return nil
}

func (r *memoryRunRepository) Update(context.Context, domain.AgentRun) error {
	return nil
}

func (r *memoryRunRepository) FindByID(_ context.Context, runID string) (domain.AgentRun, error) {
	for _, run := range r.runs {
		if run.ID == runID {
			return run, nil
		}
	}

	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *memoryRunRepository) FindLatest(context.Context) (domain.AgentRun, error) {
	if len(r.runs) == 0 {
		return domain.AgentRun{}, domain.ErrNotFound
	}

	return r.runs[len(r.runs)-1], nil
}

func (r *memoryRunRepository) FindActive(context.Context) (domain.AgentRun, error) {
	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *memoryRunRepository) List(context.Context) ([]domain.AgentRun, error) {
	return r.runs, nil
}

func (r *memoryRunRepository) ListActive(context.Context) ([]domain.AgentRun, error) {
	return nil, nil
}

func (r *memoryRunRepository) ListTerminal(context.Context) ([]domain.AgentRun, error) {
	return nil, nil
}

type memoryEventRepository struct {
	events []domain.RuntimeEvent
}

func (r *memoryEventRepository) Append(context.Context, domain.RuntimeEvent) error {
	return nil
}

func (r *memoryEventRepository) ListByRun(_ context.Context, runID string, limit int) ([]domain.RuntimeEvent, error) {
	var events []domain.RuntimeEvent
	for _, event := range r.events {
		if event.RunID == runID {
			events = append(events, event)
		}
	}
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

func (r *memoryEventRepository) ListRecent(context.Context, int) ([]domain.RuntimeEvent, error) {
	return r.events, nil
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
