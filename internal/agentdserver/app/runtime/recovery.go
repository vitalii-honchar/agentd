package runtime

import (
	"context"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RecoveryUseCase struct {
	agents     app.AgentRepository
	runtimeDBs app.RuntimeDBManager
	now        func() time.Time
}

func NewRecoveryUseCase(agents app.AgentRepository, runtimeDBs app.RuntimeDBManager) *RecoveryUseCase {
	return &RecoveryUseCase{
		agents:     agents,
		runtimeDBs: runtimeDBs,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (u *RecoveryUseCase) Recover(ctx context.Context) (RecoveryResult, error) {
	agents, err := u.agents.List(ctx)
	if err != nil {
		return RecoveryResult{}, err
	}

	recoveredAt := u.now()
	var interrupted []domain.AgentRun
	for _, agent := range agents {
		if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
			return RecoveryResult{}, err
		}
		repo := u.runtimeDBs.Runs(agent.Name)
		if repo == nil {
			continue
		}
		active, err := repo.ListActive(ctx)
		if err != nil {
			return RecoveryResult{}, err
		}
		for _, run := range active {
			run.Status = domain.AgentRunStatusInterrupted
			run.CompletedAt = &recoveredAt
			if err := repo.Update(ctx, run); err != nil {
				return RecoveryResult{}, err
			}
			interrupted = append(interrupted, run)
		}
	}

	return RecoveryResult{InterruptedRuns: interrupted, RecoveredAt: recoveredAt}, nil
}
