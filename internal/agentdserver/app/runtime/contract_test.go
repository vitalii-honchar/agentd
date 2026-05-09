package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestOutputFinalizerAcceptsValidJSON(t *testing.T) {
	t.Parallel()

	provider := &fakeStructuredOutputProvider{
		responses: []StructuredOutputResponse{{
			RequestID:  "request-1",
			OutputJSON: json.RawMessage(`{"summary":"done"}`),
		}},
	}
	validator := &fakeOutputValidator{}
	finalizer := NewOutputFinalizer(provider, validator, 1)

	result, err := finalizer.Finalize(context.Background(), OutputFinalizationRequest{
		RunID:           "run-1",
		AgentName:       "agent-a",
		RevisionID:      "rev-1",
		Model:           "gpt-5",
		OutputSchemaRaw: `{"type":"object","required":["summary"]}`,
		PlainTextResult: "done",
		History: []ProviderMessage{{
			Role:    ProviderRoleAssistant,
			Content: "done",
		}},
	})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if string(result.OutputJSON) != `{"summary":"done"}` {
		t.Fatalf("output json: got %s", result.OutputJSON)
	}
	if result.Attempts != 1 {
		t.Fatalf("attempts: got %d want 1", result.Attempts)
	}
	if provider.requests[0].OutputSchemaRaw == "" || provider.requests[0].PlainTextResult != "done" {
		t.Fatalf("provider request was not populated: %#v", provider.requests[0])
	}
}

func TestOutputFinalizerRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	provider := &fakeStructuredOutputProvider{
		responses: []StructuredOutputResponse{{
			OutputJSON: json.RawMessage(`{"summary":false}`),
		}},
	}
	validator := &fakeOutputValidator{err: domain.ErrContractOutputInvalid}
	finalizer := NewOutputFinalizer(provider, validator, 0)

	_, err := finalizer.Finalize(context.Background(), OutputFinalizationRequest{
		RunID:           "run-1",
		AgentName:       "agent-a",
		Model:           "gpt-5",
		OutputSchemaRaw: `{"type":"object","properties":{"summary":{"type":"string"}}}`,
		PlainTextResult: "done",
	})
	if !errors.Is(err, domain.ErrContractOutputInvalid) {
		t.Fatalf("Finalize error: got %v want %v", err, domain.ErrContractOutputInvalid)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider calls: got %d want 1", len(provider.requests))
	}
}

func TestOutputFinalizerBoundsRepairAttempts(t *testing.T) {
	t.Parallel()

	provider := &fakeStructuredOutputProvider{
		responses: []StructuredOutputResponse{
			{OutputJSON: json.RawMessage(`{"summary":false}`)},
			{OutputJSON: json.RawMessage(`{"summary":123}`)},
		},
	}
	validator := &fakeOutputValidator{err: domain.ErrContractOutputInvalid}
	finalizer := NewOutputFinalizer(provider, validator, 1)

	_, err := finalizer.Finalize(context.Background(), OutputFinalizationRequest{
		RunID:           "run-1",
		AgentName:       "agent-a",
		Model:           "gpt-5",
		OutputSchemaRaw: `{"type":"object","properties":{"summary":{"type":"string"}}}`,
		PlainTextResult: "done",
	})
	if !errors.Is(err, domain.ErrContractOutputInvalid) {
		t.Fatalf("Finalize error: got %v want %v", err, domain.ErrContractOutputInvalid)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider calls: got %d want 2", len(provider.requests))
	}
	if provider.requests[1].PlainTextResult != "done" {
		t.Fatalf("repair request lost plain text result: %#v", provider.requests[1])
	}
}

type fakeStructuredOutputProvider struct {
	responses []StructuredOutputResponse
	err       error
	requests  []StructuredOutputRequest
}

func (p *fakeStructuredOutputProvider) Name() string {
	return "fake"
}

func (p *fakeStructuredOutputProvider) Finalize(
	_ context.Context,
	request StructuredOutputRequest,
) (StructuredOutputResponse, error) {
	p.requests = append(p.requests, request)
	if p.err != nil {
		return StructuredOutputResponse{}, p.err
	}
	if len(p.responses) == 0 {
		return StructuredOutputResponse{}, nil
	}
	response := p.responses[0]
	p.responses = p.responses[1:]

	return response, nil
}

type fakeOutputValidator struct {
	err error
}

func (v *fakeOutputValidator) ValidateOutput(_ string, value json.RawMessage) error {
	if !json.Valid(value) {
		return domain.ErrContractOutputInvalid
	}

	return v.err
}
