package openai

import (
	"context"
	"fmt"
	"strings"

	appruntime "agentd/internal/agentdserver/app/runtime"

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
