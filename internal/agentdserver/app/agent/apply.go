package agent

import (
	"context"
	"fmt"

	"agentd/internal/agentdserver/app"
	"agentd/internal/agentdserver/domain"
)

type ApplyOutcome string

const (
	ApplyOutcomeCreated   ApplyOutcome = "created"
	ApplyOutcomeUpdated   ApplyOutcome = "updated"
	ApplyOutcomeUnchanged ApplyOutcome = "unchanged"
)

type DefinitionParser interface {
	ParseMarkdown(sourcePath string, markdown string) (domain.AgentDefinition, error)
}

type ParserFunc func(sourcePath string, markdown string) (domain.AgentDefinition, error)

func (f ParserFunc) ParseMarkdown(sourcePath string, markdown string) (domain.AgentDefinition, error) {
	return f(sourcePath, markdown)
}

type ApplyUseCase struct {
	parser     DefinitionParser
	agents     app.AgentRepository
	runtimeDBs app.RuntimeDBManager
}

type ApplyRequest struct {
	SourcePath string
	Markdown   string
}

type ApplyResult struct {
	Outcome ApplyOutcome
	Agent   domain.Agent
}

func NewApplyUseCase(
	parser DefinitionParser,
	agents app.AgentRepository,
	runtimeDBs app.RuntimeDBManager,
) (*ApplyUseCase, error) {
	if parser == nil {
		return nil, fmt.Errorf("definition parser is required")
	}
	if agents == nil {
		return nil, fmt.Errorf("agent repository is required")
	}
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}

	return &ApplyUseCase{parser: parser, agents: agents, runtimeDBs: runtimeDBs}, nil
}

func (u *ApplyUseCase) Apply(
	_ context.Context,
	_ ApplyRequest,
) (ApplyResult, error) {
	return ApplyResult{}, fmt.Errorf("apply use case not implemented")
}
