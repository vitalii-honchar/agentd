package result

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type UseCase struct {
	agents     app.AgentRepository
	runtimeDBs app.RuntimeDBManager
}

type RunResult struct {
	RunID         string
	AgentName     string
	Status        domain.AgentRunStatus
	Trigger       domain.RunTrigger
	StartedAt     *time.Time
	CompletedAt   *time.Time
	ResultFormat  domain.ResultFormat
	Result        string
	ResultJSON    json.RawMessage
	ResultSummary string
	Failure       *Failure
}

type Failure struct {
	Code    string
	Message string
}

func NewUseCase(agents app.AgentRepository, runtimeDBs app.RuntimeDBManager) (*UseCase, error) {
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}

	return &UseCase{agents: agents, runtimeDBs: runtimeDBs}, nil
}

func (u *UseCase) ResultsByAgent(ctx context.Context, agentName string) ([]RunResult, error) {
	if _, err := u.agents.FindByName(ctx, agentName); err != nil {
		return nil, err
	}
	if err := u.runtimeDBs.EnsureAgent(ctx, agentName); err != nil {
		return nil, err
	}
	repo := u.runtimeDBs.Runs(agentName)
	if repo == nil {
		return nil, domain.ErrNotFound
	}
	runs, err := repo.ListTerminal(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]RunResult, 0, len(runs))
	for _, run := range runs {
		results = append(results, toRunResult(run, false))
	}

	return results, nil
}

func (u *UseCase) ResultByRunID(ctx context.Context, runID string) (RunResult, error) {
	agents, err := u.agents.List(ctx)
	if err != nil {
		return RunResult{}, err
	}
	for _, agent := range agents {
		if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
			return RunResult{}, err
		}
		repo := u.runtimeDBs.Runs(agent.Name)
		if repo == nil {
			continue
		}
		run, err := repo.FindByID(ctx, runID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}

			return RunResult{}, err
		}
		if !run.IsTerminal() {
			return RunResult{}, domain.ErrRunNotTerminal
		}

		return toRunResult(run, true), nil
	}

	return RunResult{}, domain.ErrNotFound
}

func toRunResult(run domain.AgentRun, includeFullResult bool) RunResult {
	summary := run.ResultSummary
	if summary == "" {
		summary = Summarize(run.Result, DefaultSummaryLimit)
	}
	result := RunResult{
		RunID:         run.ID,
		AgentName:     run.AgentName,
		Status:        run.Status,
		Trigger:       run.Trigger,
		StartedAt:     run.StartedAt,
		CompletedAt:   run.CompletedAt,
		ResultFormat:  normalizedResultFormat(run.ResultFormat),
		ResultSummary: summary,
	}
	if includeFullResult {
		result.Result = run.Result
		if result.ResultFormat == domain.ResultFormatJSON && json.Valid([]byte(run.Result)) {
			result.ResultJSON = append(json.RawMessage(nil), run.Result...)
		}
	}
	if run.Status == domain.AgentRunStatusFailed || run.ErrorCode != "" || run.ErrorMessage != "" {
		result.Failure = &Failure{Code: run.ErrorCode, Message: run.ErrorMessage}
	}

	return result
}

func normalizedResultFormat(format domain.ResultFormat) domain.ResultFormat {
	if format == "" {
		return domain.ResultFormatText
	}

	return format
}
