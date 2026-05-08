package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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
	logger     *slog.Logger
	now        func() time.Time
}

type ApplyOption func(*ApplyUseCase)

func WithLogger(logger *slog.Logger) ApplyOption {
	return func(u *ApplyUseCase) {
		if logger != nil {
			u.logger = logger
		}
	}
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
	options ...ApplyOption,
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

	useCase := &ApplyUseCase{
		parser:     parser,
		agents:     agents,
		runtimeDBs: runtimeDBs,
		logger:     slog.Default(),
		now:        func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		if option != nil {
			option(useCase)
		}
	}

	return useCase, nil
}

func (u *ApplyUseCase) Apply(
	ctx context.Context,
	request ApplyRequest,
) (ApplyResult, error) {
	definition, err := u.parser.ParseMarkdown(request.SourcePath, request.Markdown)
	if err != nil {
		u.logApplyRejected(ctx, request.SourcePath, "", err)

		return ApplyResult{}, err
	}
	normalized, err := NormalizeDefinition(definition)
	if err != nil {
		u.logApplyRejected(ctx, request.SourcePath, definition.Name, err)

		return ApplyResult{}, err
	}

	existing, err := u.agents.FindByName(ctx, normalized.Definition.Name)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		u.logApplyFailed(ctx, request.SourcePath, normalized.Definition.Name, err)

		return ApplyResult{}, err
	}
	if err == nil && existing.Revision == normalized.Revision {
		u.logApplyResult(ctx, ApplyOutcomeUnchanged, existing)

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
		u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

		return ApplyResult{}, err
	}
	if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
		u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

		return ApplyResult{}, err
	}

	u.logApplyResult(ctx, outcome, agent)

	return ApplyResult{Outcome: outcome, Agent: agent}, nil
}

func (u *ApplyUseCase) logApplyResult(ctx context.Context, outcome ApplyOutcome, agent domain.Agent) {
	u.logger.InfoContext(
		ctx,
		"agent.apply."+string(outcome),
		"event", "agent.apply."+string(outcome),
		"agent", agent.Name,
		"outcome", string(outcome),
		"revision", agent.Revision,
		"source_path", agent.DefinitionSource,
		"status", string(agent.Status),
		"enabled", agent.Enabled,
		"schedule_type", string(agent.Schedule.Type),
		"vendor", agent.Vendor.Name,
		"model", agent.Vendor.Model,
	)
}

func (u *ApplyUseCase) logApplyRejected(
	ctx context.Context,
	sourcePath string,
	agentName string,
	err error,
) {
	attributes := []any{
		"event", "agent.apply.rejected",
		"outcome", "rejected",
		"source_path", sourcePath,
		"error", err,
	}
	if strings.TrimSpace(agentName) != "" {
		attributes = append(attributes, "agent", agentName)
	}

	u.logger.WarnContext(ctx, "agent.apply.rejected", attributes...)
}

func (u *ApplyUseCase) logApplyFailed(
	ctx context.Context,
	sourcePath string,
	agentName string,
	err error,
) {
	attributes := []any{
		"event", "agent.apply.failed",
		"outcome", "failed",
		"source_path", sourcePath,
		"error", err,
	}
	if strings.TrimSpace(agentName) != "" {
		attributes = append(attributes, "agent", agentName)
	}

	u.logger.ErrorContext(ctx, "agent.apply.failed", attributes...)
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
		Tools:              definition.Tools,
		MCPServers:         definition.MCPServers,
		Status:             status,
		CreatedAt:          now,
		UpdatedAt:          now,
		AppliedAt:          now,
	}
}
