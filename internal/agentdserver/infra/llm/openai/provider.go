package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

const ProviderName = "openai"

type Config struct {
	APIKey string
}

type Provider struct {
	client openaisdk.Client
}

var _ appruntime.Provider = (*Provider)(nil)
var _ appruntime.ReActProvider = (*Provider)(nil)
var _ appruntime.StructuredOutputProvider = (*Provider)(nil)

func NewProvider(cfg Config) (*Provider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("openai api key is required")
	}

	return NewProviderWithClient(openaisdk.NewClient(option.WithAPIKey(cfg.APIKey))), nil
}

func NewProviderWithClient(client openaisdk.Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) Execute(
	ctx context.Context,
	request appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	if strings.TrimSpace(request.Model) == "" {
		return appruntime.ProviderResponse{}, fmt.Errorf("openai model is required")
	}
	if strings.TrimSpace(request.Prompt) == "" {
		return appruntime.ProviderResponse{}, fmt.Errorf("openai prompt is required")
	}

	response, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{OfString: openaisdk.String(request.Prompt)},
		Model: shared.ResponsesModel(request.Model),
		Store: openaisdk.Bool(false),
	})
	if err != nil {
		return appruntime.ProviderResponse{}, fmt.Errorf("openai response: %w", err)
	}

	return appruntime.ProviderResponse{
		RequestID: response.ID,
		Output:    response.OutputText(),
		Usage: appruntime.TokenUsage{
			InputTokens:  int(response.Usage.InputTokens),
			OutputTokens: int(response.Usage.OutputTokens),
			TotalTokens:  int(response.Usage.TotalTokens),
		},
	}, nil
}

func (p *Provider) Decide(
	ctx context.Context,
	request appruntime.ReActRequest,
) (appruntime.ReActResponse, error) {
	prompt := buildReActPrompt(request)
	response, err := p.Execute(ctx, appruntime.ProviderRequest{
		RunID:     request.RunID,
		AgentName: request.AgentName,
		Model:     request.Model,
		Prompt:    prompt,
	})
	if err != nil {
		return appruntime.ReActResponse{}, err
	}

	decision := parseReActDecision(response.Output)
	decision.RequestID = response.RequestID
	decision.Usage = response.Usage

	return decision, nil
}

func (p *Provider) Finalize(
	ctx context.Context,
	request appruntime.StructuredOutputRequest,
) (appruntime.StructuredOutputResponse, error) {
	prompt := strings.Join([]string{
		"Return only JSON that matches this JSON Schema.",
		"Schema:",
		request.OutputSchemaRaw,
		"Conversation final text:",
		request.PlainTextResult,
	}, "\n")
	response, err := p.Execute(ctx, appruntime.ProviderRequest{
		RunID:     request.RunID,
		AgentName: request.AgentName,
		Model:     request.Model,
		Prompt:    prompt,
	})
	if err != nil {
		return appruntime.StructuredOutputResponse{}, err
	}
	output := json.RawMessage(strings.TrimSpace(response.Output))
	if !json.Valid(output) {
		return appruntime.StructuredOutputResponse{}, fmt.Errorf("openai structured output was not valid JSON")
	}

	return appruntime.StructuredOutputResponse{
		RequestID:  response.RequestID,
		OutputJSON: output,
		Usage:      response.Usage,
	}, nil
}

type reActDecisionOutput struct {
	Decision     string          `json:"decision"`
	ToolName     string          `json:"tool_name"`
	ToolArgsJSON json.RawMessage `json:"tool_args_json"`
	FinalText    string          `json:"final_text"`
	Failure      string          `json:"failure"`
}

func buildReActPrompt(request appruntime.ReActRequest) string {
	var builder strings.Builder
	builder.WriteString(request.Prompt)
	builder.WriteString("\n\nReturn only JSON with one of these decisions: tool_call, final, fail.")
	builder.WriteString("\nFields: decision, tool_name, tool_args_json, final_text, failure.")
	if len(request.Tools) > 0 {
		builder.WriteString("\nAvailable tools:")
		for _, tool := range request.Tools {
			builder.WriteString("\n- ")
			builder.WriteString(tool.Name)
		}
	}
	if len(request.History) > 0 {
		builder.WriteString("\n\nHistory:")
		for _, message := range request.History {
			builder.WriteString("\n")
			builder.WriteString(string(message.Role))
			builder.WriteString(": ")
			builder.WriteString(message.Content)
		}
	}

	return builder.String()
}

func parseReActDecision(output string) appruntime.ReActResponse {
	var parsed reActDecisionOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil || parsed.Decision == "" {
		return appruntime.ReActResponse{
			Decision:  "final",
			FinalText: output,
			Message:   appruntime.ProviderMessage{Role: appruntime.ProviderRoleAssistant, Content: output},
		}
	}

	response := appruntime.ReActResponse{
		Decision:     domainReActDecision(parsed.Decision),
		ToolName:     parsed.ToolName,
		ToolArgsJSON: string(parsed.ToolArgsJSON),
		FinalText:    parsed.FinalText,
		Failure:      parsed.Failure,
		Message:      appruntime.ProviderMessage{Role: appruntime.ProviderRoleAssistant, Content: output},
	}
	if response.Decision == "final" && response.FinalText == "" {
		response.FinalText = output
	}

	return response
}

func domainReActDecision(value string) domain.ReActDecision {
	switch strings.TrimSpace(value) {
	case "tool_call", "final", "fail":
		return domain.ReActDecision(value)
	default:
		return domain.ReActDecisionFinal
	}
}
