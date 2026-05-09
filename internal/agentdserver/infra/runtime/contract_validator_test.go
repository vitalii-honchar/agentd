package runtime

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestContractValidatorCompilesSchema(t *testing.T) {
	t.Parallel()

	validator := NewContractValidator()
	if _, err := validator.Compile(`{"type":"object","additionalProperties":false}`); err != nil {
		t.Fatalf("Compile valid schema: %v", err)
	}
	if _, err := validator.Compile(`{"type":`); !errors.Is(err, domain.ErrInvalidContractSchema) {
		t.Fatalf("Compile invalid schema error: got %v want %v", err, domain.ErrInvalidContractSchema)
	}
}

func TestContractValidatorValidatesInput(t *testing.T) {
	t.Parallel()

	validator := NewContractValidator()
	schema := `{"type":"object","required":["topic"],"properties":{"topic":{"type":"string"}}}`
	if err := validator.ValidateInput(schema, json.RawMessage(`{"topic":"agentd"}`)); err != nil {
		t.Fatalf("ValidateInput valid value: %v", err)
	}

	err := validator.ValidateInput(schema, json.RawMessage(`{"topic":7}`))
	if !errors.Is(err, domain.ErrContractInputInvalid) {
		t.Fatalf("ValidateInput invalid error: got %v want %v", err, domain.ErrContractInputInvalid)
	}
	if !strings.Contains(err.Error(), "topic") {
		t.Fatalf("ValidateInput diagnostic should mention topic: %v", err)
	}
}

func TestContractValidatorValidatesOutput(t *testing.T) {
	t.Parallel()

	validator := NewContractValidator()
	schema := `{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}}}`
	if err := validator.ValidateOutput(schema, json.RawMessage(`{"summary":"done"}`)); err != nil {
		t.Fatalf("ValidateOutput valid value: %v", err)
	}

	err := validator.ValidateOutput(schema, json.RawMessage(`{"summary":false}`))
	if !errors.Is(err, domain.ErrContractOutputInvalid) {
		t.Fatalf("ValidateOutput invalid error: got %v want %v", err, domain.ErrContractOutputInvalid)
	}
	if !strings.Contains(err.Error(), "summary") {
		t.Fatalf("ValidateOutput diagnostic should mention summary: %v", err)
	}
}
