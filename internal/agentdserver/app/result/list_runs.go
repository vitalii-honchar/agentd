package result

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type ListRunsUseCase struct {
	agents     app.AgentRepository
	runtimeDBs app.RuntimeDBManager
}

func NewListRunsUseCase(agents app.AgentRepository, runtimeDBs app.RuntimeDBManager) (*ListRunsUseCase, error) {
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}

	return &ListRunsUseCase{agents: agents, runtimeDBs: runtimeDBs}, nil
}

func (u *ListRunsUseCase) ListRuns(ctx context.Context, includeAll bool) ([]domain.AgentRun, error) {
	agents, err := u.agents.List(ctx)
	if err != nil {
		return nil, err
	}

	var runs []domain.AgentRun
	for _, agent := range agents {
		repo := u.runtimeDBs.Runs(agent.Name)
		if repo == nil {
			continue
		}
		var agentRuns []domain.AgentRun
		if includeAll {
			agentRuns, err = repo.List(ctx)
		} else {
			agentRuns, err = repo.ListActive(ctx)
		}
		if err != nil {
			return nil, err
		}
		runs = append(runs, agentRuns...)
	}
	sort.SliceStable(runs, func(i, j int) bool {
		return runSortTime(runs[i]).After(runSortTime(runs[j]))
	})

	return runs, nil
}

func runSortTime(run domain.AgentRun) time.Time {
	switch {
	case run.StartedAt != nil:
		return *run.StartedAt
	case run.DueAt != nil:
		return *run.DueAt
	case run.CompletedAt != nil:
		return *run.CompletedAt
	default:
		return time.Time{}
	}
}
