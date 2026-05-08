package runtime

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type Provider interface {
	Name() string
	Execute(ctx context.Context, request ProviderRequest) (ProviderResponse, error)
}

type ToolExecutor interface {
	Execute(ctx context.Context, request ToolRequest) (ToolResult, error)
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

type ToolRequest struct {
	RunID   string
	Agent   domain.Agent
	Tool    domain.ToolPermission
	WorkDir string
}

type ToolResult struct {
	StdoutSummary string
	StderrSummary string
	ExitCode      int
	TimedOut      bool
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
