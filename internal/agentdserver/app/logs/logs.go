package logs

import (
	"context"
	"fmt"

	"agentd/internal/agentdserver/app"
	"agentd/internal/agentdserver/domain"
)

type Query struct {
	AgentName string
	RunID     string
	Tail      int
}

type Result struct {
	Agent   domain.Agent
	Run     domain.AgentRun
	Entries []app.LogEntry
}

type UseCase struct {
	agents     app.AgentRepository
	runtimeDBs app.RuntimeDBManager
	reader     app.RunLogReader
}

func NewUseCase(
	agents app.AgentRepository,
	runtimeDBs app.RuntimeDBManager,
	reader app.RunLogReader,
) (*UseCase, error) {
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}
	if reader == nil {
		return nil, fmt.Errorf("run log reader is required")
	}

	return &UseCase{agents: agents, runtimeDBs: runtimeDBs, reader: reader}, nil
}

func (u *UseCase) Read(ctx context.Context, query Query) (Result, error) {
	agent, err := u.agents.FindByName(ctx, query.AgentName)
	if err != nil {
		return Result{}, err
	}

	repo := u.runtimeDBs.Runs(agent.Name)
	if repo == nil {
		return Result{}, domain.ErrNotFound
	}

	run, err := findRun(ctx, repo, query.RunID)
	if err != nil {
		return Result{}, err
	}

	entries, err := u.reader.Read(ctx, app.LogQuery{
		AgentName: agent.Name,
		RunID:     run.ID,
		LogPath:   run.LogPath,
		Tail:      query.Tail,
	})
	if err != nil {
		return Result{}, err
	}

	return Result{Agent: agent, Run: run, Entries: entries}, nil
}

func findRun(ctx context.Context, repo app.AgentRunRepository, runID string) (domain.AgentRun, error) {
	if runID != "" {
		return repo.FindByID(ctx, runID)
	}

	return repo.FindLatest(ctx)
}
