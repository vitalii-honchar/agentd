package scheduling

import (
	"context"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type Scheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Reconcile(ctx context.Context, agents []domain.Agent, handler Handler) error
	Unschedule(agentName string) error
	NextRun(schedule domain.Schedule, from time.Time) (*time.Time, error)
}

type Handler func(context.Context, Trigger) error

type Trigger struct {
	AgentName string
	DueAt     time.Time
	Source    domain.RunTrigger
}
