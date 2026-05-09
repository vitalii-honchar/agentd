package agent

import (
	"context"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type InspectUseCase struct {
	agents app.AgentRepository
}

func NewInspectUseCase(agents app.AgentRepository) (*InspectUseCase, error) {
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}

	return &InspectUseCase{agents: agents}, nil
}

func (u *InspectUseCase) Inspect(ctx context.Context, name string) (domain.Agent, error) {
	agent, err := u.agents.FindByName(ctx, name)
	if err != nil {
		return domain.Agent{}, err
	}
	agent.Contract = maskContractForInspect(agent.Contract)

	return agent, nil
}

func maskContractForInspect(contract *domain.AgentContract) *domain.AgentContract {
	if contract == nil {
		return nil
	}
	masked := *contract
	masked.InputSchemaRaw = ""
	masked.OutputSchemaRaw = ""

	return &masked
}

type ListUseCase struct {
	agents app.AgentRepository
}

func NewListUseCase(agents app.AgentRepository) (*ListUseCase, error) {
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}

	return &ListUseCase{agents: agents}, nil
}

func (u *ListUseCase) List(ctx context.Context) ([]domain.Agent, error) {
	return u.agents.List(ctx)
}
