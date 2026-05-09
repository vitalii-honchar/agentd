package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	goagentllm "github.com/vitalii-honchar/go-agent/pkg/goagent/llm"
)

var ErrReActFailed = errors.New("react execution failed")

type ReActAdapter struct {
	provider ReActProvider
	base     ReActRequest
}

func NewReActAdapter(provider ReActProvider) *ReActAdapter {
	return &ReActAdapter{provider: provider}
}

func NewReActAdapterWithRequest(provider ReActProvider, base ReActRequest) *ReActAdapter {
	return &ReActAdapter{provider: provider, base: base}
}

func (a *ReActAdapter) CallDynamic(
	ctx context.Context,
	messages []goagentllm.DynamicMessage,
	tools []goagentllm.DynamicTool,
) (goagentllm.DynamicMessage, error) {
	if a == nil || a.provider == nil {
		return goagentllm.DynamicMessage{}, fmt.Errorf("react provider is required")
	}
	request := a.base
	request.History = dynamicHistory(messages)
	request.Tools = dynamicTools(tools)
	response, err := a.provider.Decide(ctx, request)
	if err != nil {
		return goagentllm.DynamicMessage{}, err
	}

	return dynamicMessageFromDecision(response)
}

func (a *ReActAdapter) CallWithDynamicStructuredOutput(
	_ context.Context,
	messages []goagentllm.DynamicMessage,
	_ json.RawMessage,
) (json.RawMessage, error) {
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message.Role != goagentllm.DynamicRoleAssistant || message.Content == "" {
			continue
		}
		body, err := json.Marshal(map[string]string{"final_text": message.Content})
		if err != nil {
			return nil, err
		}

		return body, nil
	}

	return nil, fmt.Errorf("%w: final assistant message is missing", ErrReActFailed)
}

func dynamicMessageFromDecision(response ReActResponse) (goagentllm.DynamicMessage, error) {
	switch response.Decision {
	case domain.ReActDecisionToolCall:
		callID := response.RequestID
		if callID == "" {
			callID = "call-" + response.ToolName
		}

		return goagentllm.DynamicMessage{
			Role:    goagentllm.DynamicRoleAssistant,
			Content: response.Message.Content,
			ToolCalls: []goagentllm.DynamicToolCall{{
				ID:       callID,
				ToolName: response.ToolName,
				ArgsJSON: json.RawMessage(response.ToolArgsJSON),
			}},
		}, nil
	case domain.ReActDecisionFinal:
		content := response.FinalText
		if content == "" {
			content = response.Message.Content
		}

		return goagentllm.DynamicMessage{
			Role:    goagentllm.DynamicRoleAssistant,
			Content: content,
			End:     true,
		}, nil
	case domain.ReActDecisionFail:
		if response.Failure == "" {
			response.Failure = string(domain.ReActDecisionFail)
		}

		return goagentllm.DynamicMessage{}, fmt.Errorf("%w: %s", ErrReActFailed, response.Failure)
	default:
		return goagentllm.DynamicMessage{}, fmt.Errorf("%w: unsupported decision %q", ErrReActFailed, response.Decision)
	}
}

func dynamicHistory(messages []goagentllm.DynamicMessage) []ProviderMessage {
	history := make([]ProviderMessage, 0, len(messages))
	for _, message := range messages {
		history = append(history, ProviderMessage{
			Role:    providerRoleFromDynamic(message.Role),
			Content: message.Content,
		})
	}

	return history
}

func providerRoleFromDynamic(role goagentllm.DynamicRole) ProviderRole {
	switch role {
	case goagentllm.DynamicRoleSystem:
		return ProviderRoleSystem
	case goagentllm.DynamicRoleAssistant:
		return ProviderRoleAssistant
	case goagentllm.DynamicRoleTool:
		return ProviderRoleTool
	default:
		return ProviderRoleUser
	}
}

func dynamicTools(tools []goagentllm.DynamicTool) []domain.ToolPermission {
	mapped := make([]domain.ToolPermission, 0, len(tools))
	for _, tool := range tools {
		mapped = append(mapped, domain.ToolPermission{
			Name: tool.Name,
		})
	}

	return mapped
}
