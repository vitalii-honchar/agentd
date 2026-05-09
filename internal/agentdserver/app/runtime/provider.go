package runtime

import (
	"context"
	"encoding/json"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type Provider interface {
	Name() string
	Execute(ctx context.Context, request ProviderRequest) (ProviderResponse, error)
}

type ReActProvider interface {
	Name() string
	Decide(ctx context.Context, request ReActRequest) (ReActResponse, error)
}

type StructuredOutputProvider interface {
	Name() string
	Finalize(ctx context.Context, request StructuredOutputRequest) (StructuredOutputResponse, error)
}

type ToolExecutor interface {
	Execute(ctx context.Context, request ToolRequest) (ToolResult, error)
}

type ProviderRole string

const (
	ProviderRoleSystem    ProviderRole = "system"
	ProviderRoleUser      ProviderRole = "user"
	ProviderRoleAssistant ProviderRole = "assistant"
	ProviderRoleTool      ProviderRole = "tool"
)

type ProviderRequest struct {
	RunID      string
	AgentName  string
	Model      string
	Prompt     string
	Tools      []domain.ToolPermission
	MCPServers []domain.ToolPermission
	Access     domain.AccessPolicy
}

type ProviderMessage struct {
	Role       ProviderRole
	Content    string
	ToolName   string
	ToolCallID string
}

type ReActRequest struct {
	RunID      string
	AgentName  string
	RevisionID string
	Model      string
	Prompt     string
	History    []ProviderMessage
	Tools      []domain.ToolPermission
	MCPServers []domain.ToolPermission
	Access     domain.AccessPolicy
	MaxSteps   int
	StepIndex  int
}

type ReActResponse struct {
	RequestID    string
	Decision     domain.ReActDecision
	Message      ProviderMessage
	ToolName     string
	ToolArgsJSON string
	FinalText    string
	Failure      string
	Usage        TokenUsage
}

type StructuredOutputRequest struct {
	RunID           string
	AgentName       string
	RevisionID      string
	Model           string
	OutputSchemaRaw string
	History         []ProviderMessage
	PlainTextResult string
}

type StructuredOutputResponse struct {
	RequestID  string
	OutputJSON json.RawMessage
	Usage      TokenUsage
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
	ResultSummary string
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
