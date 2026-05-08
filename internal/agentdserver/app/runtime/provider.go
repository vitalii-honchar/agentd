package runtime

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type Provider interface {
	Name() string
	Execute(ctx context.Context, request ProviderRequest) (ProviderResponse, error)
}

type ProviderRequest struct {
	RunID      string
	AgentName  string
	Model      string
	Prompt     string
	Tools      []domain.ToolPermission
	MCPServers []domain.ToolPermission
	Access     domain.AccessPolicy
}

type ProviderResponse struct {
	RequestID string
	Output    string
	Usage     TokenUsage
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
