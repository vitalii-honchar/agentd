package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrConflict          = errors.New("conflict")
	ErrInvalidDefinition = errors.New("invalid agent definition")
	ErrInvalidState      = errors.New("invalid state")
	ErrAgentDisabled     = errors.New("agent disabled")
	ErrRunAlreadyActive  = errors.New("agent run already active")
	ErrUnsupportedVendor = errors.New("unsupported llm vendor")
)

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
