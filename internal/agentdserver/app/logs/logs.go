package logs

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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
	if query.RunID == "" {
		return Result{}, fmt.Errorf("run ID is required")
	}

	agents, err := u.agents.List(ctx)
	if err != nil {
		return Result{}, err
	}
	for _, agent := range agents {
		if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
			return Result{}, err
		}
		repo := u.runtimeDBs.Runs(agent.Name)
		if repo == nil {
			continue
		}

		run, err := repo.FindByID(ctx, query.RunID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return Result{}, err
		}

		return u.readRunLogs(ctx, agent, run, query.Tail)
	}

	return Result{}, domain.ErrNotFound
}

func (u *UseCase) readRunLogs(
	ctx context.Context,
	agent domain.Agent,
	run domain.AgentRun,
	tail int,
) (Result, error) {
	entries, err := u.reader.Read(ctx, app.LogQuery{
		AgentName: agent.Name,
		RunID:     run.ID,
		LogPath:   run.LogPath,
		Tail:      tail,
	})
	if err != nil {
		return Result{}, err
	}
	if events := u.runtimeDBs.Events(agent.Name); events != nil {
		actionEvents, err := events.ListByRun(ctx, run.ID, tail)
		if err != nil {
			return Result{}, err
		}
		entries = append(runtimeEventsToLogEntries(actionEvents), entries...)
		if tail > 0 && len(entries) > tail {
			entries = entries[:tail]
		}
	}

	return Result{Agent: agent, Run: run, Entries: entries}, nil
}

func runtimeEventsToLogEntries(events []domain.RuntimeEvent) []app.LogEntry {
	entries := make([]app.LogEntry, 0, len(events))
	for _, event := range events {
		entries = append(entries, app.LogEntry{
			Timestamp: event.CreatedAt,
			RunID:     event.RunID,
			Action:    event.EventType,
			Message:   event.Message,
			Line:      event.Message,
		})
	}

	return entries
}
