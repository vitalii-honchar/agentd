package scheduling

import (
	"context"

	"agentd/internal/agentdserver/app"
)

type ReconcileUseCase struct {
	agents    app.AgentRepository
	scheduler Scheduler
	handler   Handler
}

func NewReconcileUseCase(
	agents app.AgentRepository,
	scheduler Scheduler,
	handler Handler,
) *ReconcileUseCase {
	return &ReconcileUseCase{agents: agents, scheduler: scheduler, handler: handler}
}

func (u *ReconcileUseCase) Reconcile(ctx context.Context) error {
	agents, err := u.agents.List(ctx)
	if err != nil {
		return err
	}

	return u.scheduler.Reconcile(ctx, agents, u.handler)
}
