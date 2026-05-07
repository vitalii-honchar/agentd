package runtime

import (
	"context"

	"agentd/internal/agentdserver/domain"
)

type StopUseCase struct {
	manager Manager
}

func NewStopUseCase(manager Manager) *StopUseCase {
	return &StopUseCase{manager: manager}
}

func (u *StopUseCase) Stop(ctx context.Context, agentName, runID string) (domain.AgentRun, error) {
	return u.manager.Stop(ctx, StopRequest{AgentName: agentName, RunID: runID})
}
