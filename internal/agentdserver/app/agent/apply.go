package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	now        func() time.Time
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

	return &ApplyUseCase{
		parser:     parser,
		agents:     agents,
		runtimeDBs: runtimeDBs,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

func (u *ApplyUseCase) Apply(
	ctx context.Context,
	request ApplyRequest,
) (ApplyResult, error) {
	definition, err := u.parser.ParseMarkdown(request.SourcePath, request.Markdown)
	if err != nil {
		return ApplyResult{}, err
	}
	normalized, err := NormalizeDefinition(definition)
	if err != nil {
		return ApplyResult{}, err
	}

	existing, err := u.agents.FindByName(ctx, normalized.Definition.Name)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return ApplyResult{}, err
	}
	if err == nil && existing.Revision == normalized.Revision {
		return ApplyResult{Outcome: ApplyOutcomeUnchanged, Agent: existing}, nil
	}

	agent := agentFromDefinition(normalized.Definition, normalized.Revision, u.now())
	outcome := ApplyOutcomeCreated
	if err == nil {
		agent.CreatedAt = existing.CreatedAt
		agent.LastRunID = existing.LastRunID
		agent.LastError = existing.LastError
		outcome = ApplyOutcomeUpdated
	}

	if err := u.agents.Save(ctx, agent, normalized.Definition.Tools, normalized.Definition.MCPServers); err != nil {
		return ApplyResult{}, err
	}
	if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
		return ApplyResult{}, err
	}

	return ApplyResult{Outcome: outcome, Agent: agent}, nil
}

func agentFromDefinition(
	definition domain.AgentDefinition,
	revision string,
	now time.Time,
) domain.Agent {
	status := domain.AgentStatusActive
	if !definition.Enabled {
		status = domain.AgentStatusDisabled
	}

	return domain.Agent{
		Name:               definition.Name,
		Revision:           revision,
		DefinitionSource:   definition.SourcePath,
		DefinitionMarkdown: definition.RawMarkdown,
		Prompt:             definition.Prompt,
		Enabled:            definition.Enabled,
		Vendor:             definition.Vendor,
		Schedule:           definition.Schedule,
		Status:             status,
		CreatedAt:          now,
		UpdatedAt:          now,
		AppliedAt:          now,
	}
}
