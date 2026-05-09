package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound              = errors.New("not found")
	ErrAlreadyExists         = errors.New("already exists")
	ErrConflict              = errors.New("conflict")
	ErrInvalidDefinition     = errors.New("invalid agent definition")
	ErrInvalidState          = errors.New("invalid state")
	ErrAgentDisabled         = errors.New("agent disabled")
	ErrRunAlreadyActive      = errors.New("agent run already active")
	ErrRunNotTerminal        = errors.New("agent run is not terminal")
	ErrUnsupportedVendor     = errors.New("unsupported llm vendor")
	ErrInvalidContractSchema = errors.New("invalid agent contract schema")
	ErrContractInputInvalid  = errors.New("contract input invalid")
	ErrContractOutputInvalid = errors.New("contract output invalid")
	ErrProviderUnavailable   = errors.New("llm provider unavailable")
	ErrProviderRequestFailed = errors.New("llm provider request failed")
)

type ErrorCode string

const (
	ErrorCodeInternal              ErrorCode = "internal_error"
	ErrorCodeNotFound              ErrorCode = "not_found"
	ErrorCodeAlreadyExists         ErrorCode = "already_exists"
	ErrorCodeConflict              ErrorCode = "conflict"
	ErrorCodeInvalidDefinition     ErrorCode = "invalid_definition"
	ErrorCodeInvalidState          ErrorCode = "invalid_state"
	ErrorCodeAgentDisabled         ErrorCode = "agent_disabled"
	ErrorCodeRunAlreadyActive      ErrorCode = "run_already_active"
	ErrorCodeRunNotTerminal        ErrorCode = "run_not_terminal"
	ErrorCodeUnsupportedVendor     ErrorCode = "unsupported_vendor"
	ErrorCodeContractSchemaInvalid ErrorCode = "contract_schema_invalid"
	ErrorCodeContractInputInvalid  ErrorCode = "contract_input_invalid"
	ErrorCodeContractOutputInvalid ErrorCode = "contract_output_invalid"
	ErrorCodeProviderUnavailable   ErrorCode = "provider_unavailable"
	ErrorCodeProviderRequestFailed ErrorCode = "provider_request_failed"
)

func ErrorCodeFor(err error) ErrorCode {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotFound):
		return ErrorCodeNotFound
	case errors.Is(err, ErrAlreadyExists):
		return ErrorCodeAlreadyExists
	case errors.Is(err, ErrConflict):
		return ErrorCodeConflict
	case errors.Is(err, ErrInvalidDefinition):
		return ErrorCodeInvalidDefinition
	case errors.Is(err, ErrInvalidState):
		return ErrorCodeInvalidState
	case errors.Is(err, ErrAgentDisabled):
		return ErrorCodeAgentDisabled
	case errors.Is(err, ErrRunAlreadyActive):
		return ErrorCodeRunAlreadyActive
	case errors.Is(err, ErrRunNotTerminal):
		return ErrorCodeRunNotTerminal
	case errors.Is(err, ErrUnsupportedVendor):
		return ErrorCodeUnsupportedVendor
	case errors.Is(err, ErrInvalidContractSchema):
		return ErrorCodeContractSchemaInvalid
	case errors.Is(err, ErrContractInputInvalid):
		return ErrorCodeContractInputInvalid
	case errors.Is(err, ErrContractOutputInvalid):
		return ErrorCodeContractOutputInvalid
	case errors.Is(err, ErrProviderUnavailable):
		return ErrorCodeProviderUnavailable
	case errors.Is(err, ErrProviderRequestFailed):
		return ErrorCodeProviderRequestFailed
	default:
		return ErrorCodeInternal
	}
}

type ValidationIssue struct {
	Field   string
	Message string
}

type ValidationError struct {
	Issues []ValidationIssue
}

func NewValidationError(issues []ValidationIssue) *ValidationError {
	copied := make([]ValidationIssue, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(issue.Message) == "" {
			continue
		}
		copied = append(copied, issue)
	}

	return &ValidationError{Issues: copied}
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ErrInvalidDefinition.Error()
	}

	first := e.Issues[0]
	if strings.TrimSpace(first.Field) == "" {
		return fmt.Sprintf("%s: %s", ErrInvalidDefinition, first.Message)
	}

	return fmt.Sprintf("%s: %s: %s", ErrInvalidDefinition, first.Field, first.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidDefinition
}
