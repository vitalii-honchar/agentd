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
	return u.agents.FindByName(ctx, name)
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
