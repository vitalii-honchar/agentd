package runtime

import (
	"context"

	"agentd/internal/agentdserver/app"
	"agentd/internal/agentdserver/domain"
)

type ExecuteUseCase struct {
	agents  app.AgentRepository
	manager Manager
}

func NewExecuteUseCase(agents app.AgentRepository, manager Manager) *ExecuteUseCase {
	return &ExecuteUseCase{agents: agents, manager: manager}
}

func (u *ExecuteUseCase) Execute(ctx context.Context, agentName string) (domain.AgentRun, error) {
	agent, err := u.agents.FindByName(ctx, agentName)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if err := agent.CanExecute(); err != nil {
		return domain.AgentRun{}, err
	}

	return u.manager.Execute(ctx, ExecuteRequest{
		Agent:   agent,
		Trigger: domain.RunTriggerManual,
	})
}
