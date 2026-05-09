package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type OutputContractValidator interface {
	ValidateOutput(schemaRaw string, value json.RawMessage) error
}

type OutputFinalizer struct {
	provider          StructuredOutputProvider
	validator         OutputContractValidator
	maxRepairAttempts int
}

type OutputFinalizationRequest struct {
	RunID           string
	AgentName       string
	RevisionID      string
	Model           string
	OutputSchemaRaw string
	History         []ProviderMessage
	PlainTextResult string
}

type OutputFinalizationResult struct {
	RequestID  string
	OutputJSON json.RawMessage
	Attempts   int
	Usage      TokenUsage
}

func NewOutputFinalizer(
	provider StructuredOutputProvider,
	validator OutputContractValidator,
	maxRepairAttempts int,
) *OutputFinalizer {
	if maxRepairAttempts < 0 {
		maxRepairAttempts = 0
	}

	return &OutputFinalizer{
		provider:          provider,
		validator:         validator,
		maxRepairAttempts: maxRepairAttempts,
	}
}

func (f *OutputFinalizer) Finalize(
	ctx context.Context,
	request OutputFinalizationRequest,
) (OutputFinalizationResult, error) {
	if f == nil || f.provider == nil {
		return OutputFinalizationResult{}, fmt.Errorf("structured output provider is required")
	}
	if f.validator == nil {
		return OutputFinalizationResult{}, fmt.Errorf("contract output validator is required")
	}
	if request.OutputSchemaRaw == "" {
		return OutputFinalizationResult{}, fmt.Errorf("%w: contract.output is required", domain.ErrInvalidContractSchema)
	}

	attempts := f.maxRepairAttempts + 1
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		response, err := f.provider.Finalize(ctx, StructuredOutputRequest{
			RunID:           request.RunID,
			AgentName:       request.AgentName,
			RevisionID:      request.RevisionID,
			Model:           request.Model,
			OutputSchemaRaw: request.OutputSchemaRaw,
			History:         append([]ProviderMessage(nil), request.History...),
			PlainTextResult: request.PlainTextResult,
		})
		if err != nil {
			return OutputFinalizationResult{}, err
		}
		if err := f.validator.ValidateOutput(request.OutputSchemaRaw, response.OutputJSON); err != nil {
			lastErr = err

			continue
		}

		return OutputFinalizationResult{
			RequestID:  response.RequestID,
			OutputJSON: append(json.RawMessage(nil), response.OutputJSON...),
			Attempts:   attempt,
			Usage:      response.Usage,
		}, nil
	}
	if lastErr == nil {
		lastErr = domain.ErrContractOutputInvalid
	}

	return OutputFinalizationResult{}, lastErr
}
