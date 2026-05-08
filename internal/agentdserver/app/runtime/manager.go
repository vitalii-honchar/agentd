package runtime

import (
	"context"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type Manager interface {
	Execute(ctx context.Context, request ExecuteRequest) (domain.AgentRun, error)
	Stop(ctx context.Context, request StopRequest) (domain.AgentRun, error)
	Recover(ctx context.Context) (RecoveryResult, error)
	ActiveRuns(ctx context.Context) ([]domain.AgentRun, error)
}

type ExecuteRequest struct {
	Agent   domain.Agent
	Trigger domain.RunTrigger
	DueAt   *time.Time
	Inputs  map[string]string
}

type StopRequest struct {
	AgentName string
	RunID     string
}

type RecoveryResult struct {
	InterruptedRuns []domain.AgentRun
	RecoveredAt     time.Time
}

type ActiveRunTracker interface {
	Add(run domain.AgentRun) error
	Remove(runID string)
	FindByAgent(agentName string) (domain.AgentRun, bool)
	List() []domain.AgentRun
}
