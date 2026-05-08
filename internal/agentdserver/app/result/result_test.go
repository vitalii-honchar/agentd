package result

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestUseCaseResultsByAgentReturnsTerminalRuns(t *testing.T) {
	t.Parallel()

	completedAt := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	useCase := newResultUseCaseForTest(t, []domain.AgentRun{{
		ID:            "run-1",
		AgentName:     "hacker-news-builder-brief",
		Status:        domain.AgentRunStatusCompleted,
		Trigger:       domain.RunTriggerSchedule,
		CompletedAt:   &completedAt,
		Result:        "full result",
		ResultSummary: "summary",
	}})

	results, err := useCase.ResultsByAgent(context.Background(), "hacker-news-builder-brief")
	if err != nil {
		t.Fatalf("ResultsByAgent: %v", err)
	}
	if len(results) != 1 || results[0].RunID != "run-1" || results[0].ResultSummary != "summary" {
		t.Fatalf("results: %#v", results)
	}
}

func TestUseCaseResultByRunIDRejectsActiveRun(t *testing.T) {
	t.Parallel()

	useCase := newResultUseCaseForTest(t, []domain.AgentRun{{
		ID:        "run-active",
		AgentName: "website-snapshot-analyst",
		Status:    domain.AgentRunStatusRunning,
		Trigger:   domain.RunTriggerManual,
	}})

	_, err := useCase.ResultByRunID(context.Background(), "run-active")
	if !errors.Is(err, domain.ErrRunNotTerminal) {
		t.Fatalf("ResultByRunID error: got %v want ErrRunNotTerminal", err)
	}
}

func TestUseCaseResultByRunIDReturnsFullResult(t *testing.T) {
	t.Parallel()

	completedAt := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	useCase := newResultUseCaseForTest(t, []domain.AgentRun{{
		ID:            "run-2",
		AgentName:     "website-snapshot-analyst",
		Status:        domain.AgentRunStatusFailed,
		Trigger:       domain.RunTriggerManual,
		CompletedAt:   &completedAt,
		Result:        "failed with captured stderr",
		ResultSummary: "failed",
		ErrorCode:     "tool_failed",
		ErrorMessage:  "tool exited 1",
	}})

	result, err := useCase.ResultByRunID(context.Background(), "run-2")
	if err != nil {
		t.Fatalf("ResultByRunID: %v", err)
	}
	if result.Result != "failed with captured stderr" || result.Failure == nil {
		t.Fatalf("result: %#v", result)
	}
}

func newResultUseCaseForTest(t *testing.T, runs []domain.AgentRun) *UseCase {
	t.Helper()

	useCase, err := NewUseCase(
		&resultAgentRepo{agents: []domain.Agent{{Name: "hacker-news-builder-brief"}, {Name: "website-snapshot-analyst"}}},
		&resultRuntimeDBs{runs: &resultRunRepo{runs: runs}},
	)
	if err != nil {
		t.Fatalf("NewUseCase: %v", err)
	}

	return useCase
}

type resultAgentRepo struct {
	agents []domain.Agent
}

func (r *resultAgentRepo) Save(context.Context, domain.Agent, []domain.ToolPermission, []domain.ToolPermission) error {
	return nil
}

func (r *resultAgentRepo) FindByName(_ context.Context, name string) (domain.Agent, error) {
	for _, agent := range r.agents {
		if agent.Name == name {
			return agent, nil
		}
	}

	return domain.Agent{}, domain.ErrNotFound
}

func (r *resultAgentRepo) List(context.Context) ([]domain.Agent, error) {
	return r.agents, nil
}

type resultRuntimeDBs struct {
	runs app.AgentRunRepository
}

func (r *resultRuntimeDBs) EnsureAgent(context.Context, string) error { return nil }
func (r *resultRuntimeDBs) Runs(string) app.AgentRunRepository        { return r.runs }
func (r *resultRuntimeDBs) Events(string) app.RuntimeEventRepository  { return nil }
func (r *resultRuntimeDBs) Close(context.Context) error               { return nil }

type resultRunRepo struct {
	runs []domain.AgentRun
}

func (r *resultRunRepo) Create(context.Context, domain.AgentRun) error { return nil }
func (r *resultRunRepo) Update(context.Context, domain.AgentRun) error { return nil }

func (r *resultRunRepo) FindByID(_ context.Context, runID string) (domain.AgentRun, error) {
	for _, run := range r.runs {
		if run.ID == runID {
			return run, nil
		}
	}

	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *resultRunRepo) FindLatest(context.Context) (domain.AgentRun, error) {
	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *resultRunRepo) FindActive(context.Context) (domain.AgentRun, error) {
	return domain.AgentRun{}, domain.ErrNotFound
}

func (r *resultRunRepo) List(context.Context) ([]domain.AgentRun, error) {
	return r.runs, nil
}

func (r *resultRunRepo) ListActive(context.Context) ([]domain.AgentRun, error) {
	return nil, nil
}

func (r *resultRunRepo) ListTerminal(context.Context) ([]domain.AgentRun, error) {
	var terminal []domain.AgentRun
	for _, run := range r.runs {
		if run.IsTerminal() {
			terminal = append(terminal, run)
		}
	}

	return terminal, nil
}
