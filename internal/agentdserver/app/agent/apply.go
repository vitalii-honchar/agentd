package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"github.com/google/uuid"
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
	revisions  app.AgentRevisionRepository
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
	Outcome        ApplyOutcome
	Agent          domain.Agent
	RevisionID     string
	ArtifactPath   string
	RevisionStatus domain.AgentRevisionStatus
	RevisionReused bool
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
	revisions, ok := agents.(app.AgentRevisionRepository)
	if !ok {
		return nil, fmt.Errorf("agent repository must support revisions")
	}

	useCase := &ApplyUseCase{
		parser:     parser,
		agents:     agents,
		revisions:  revisions,
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
	if err == nil {
		revision, revisionErr := u.revisions.FindRevisionByDigest(ctx, existing.Name, normalized.Revision)
		if revisionErr != nil && !errors.Is(revisionErr, domain.ErrNotFound) {
			u.logApplyFailed(ctx, request.SourcePath, normalized.Definition.Name, revisionErr)

			return ApplyResult{}, revisionErr
		}
		if revisionErr == nil {
			existing.Revision = revision.RevisionID
			u.logApplyResult(ctx, ApplyOutcomeUnchanged, existing)

			return applyResult(ApplyOutcomeUnchanged, existing, revision, true), nil
		}
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
	revision, err := u.revisions.FindRevisionByDigest(ctx, agent.Name, normalized.Revision)
	reusedRevision := err == nil
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

			return ApplyResult{}, err
		}
		revision, err = revisionFromDefinition(normalized.Definition, uuid.NewString(), normalized.Revision, u.now())
		if err != nil {
			u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

			return ApplyResult{}, err
		}
		if err := u.revisions.SaveRevision(ctx, revision); err != nil {
			u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

			return ApplyResult{}, err
		}
	}
	agent.Revision = revision.RevisionID
	if err := u.runtimeDBs.EnsureAgent(ctx, agent.Name); err != nil {
		u.logApplyFailed(ctx, request.SourcePath, agent.Name, err)

		return ApplyResult{}, err
	}

	u.logApplyResult(ctx, outcome, agent)

	return applyResult(outcome, agent, revision, reusedRevision), nil
}

func applyResult(
	outcome ApplyOutcome,
	agent domain.Agent,
	revision domain.AgentRevision,
	reused bool,
) ApplyResult {
	return ApplyResult{
		Outcome:        outcome,
		Agent:          agent,
		RevisionID:     revision.RevisionID,
		ArtifactPath:   revision.ArtifactPath,
		RevisionStatus: revision.Status,
		RevisionReused: reused,
	}
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

func revisionFromDefinition(
	definition domain.AgentDefinition,
	revisionID string,
	contentDigest string,
	now time.Time,
) (domain.AgentRevision, error) {
	finalizedAt := now
	environment, artifactFiles, err := captureDefinitionEnvironment(definition, revisionID, now)
	if err != nil {
		return domain.AgentRevision{}, err
	}

	return domain.AgentRevision{
		AgentName:       definition.Name,
		RevisionID:      revisionID,
		ContentDigest:   contentDigest,
		SourcePath:      definition.SourcePath,
		ArtifactPath:    filepath.Join("data", "work", definition.Name, revisionID),
		EnvironmentJSON: "[]",
		Prompt:          definition.Prompt,
		Vendor:          definition.Vendor,
		Schedule:        definition.Schedule,
		Status:          domain.AgentRevisionStatusFinalized,
		CreatedAt:       now,
		FinalizedAt:     &finalizedAt,
		Tools:           revisionToolsFromDefinition(definition, revisionID, now),
		ArtifactFiles:   artifactFiles,
		Environment:     environment,
	}, nil
}

func revisionToolsFromDefinition(
	definition domain.AgentDefinition,
	revisionID string,
	now time.Time,
) []domain.RevisionTool {
	tools := make([]domain.RevisionTool, 0, len(definition.Tools))
	for _, tool := range definition.Tools {
		kind := tool.Kind
		if kind == domain.ToolKindLocalTool {
			kind = domain.ToolKindCustomTool
		}
		revisionTool := domain.RevisionTool{
			AgentName:       definition.Name,
			RevisionID:      revisionID,
			Name:            tool.Name,
			Kind:            kind,
			OriginalCommand: tool.Command,
			Args:            append([]string(nil), tool.Args...),
			Env:             append([]string(nil), tool.Env...),
			Timeout:         tool.Timeout,
			ReadPaths:       append([]string(nil), tool.ReadPaths...),
			WritePaths:      append([]string(nil), tool.WritePaths...),
			NetworkAllow:    append([]string(nil), tool.NetworkAllow...),
			CreatedAt:       now,
		}
		if kind == domain.ToolKindHostTool {
			revisionTool.HostCommand = tool.Command
		}
		tools = append(tools, revisionTool)
	}

	return tools
}

func captureDefinitionEnvironment(
	definition domain.AgentDefinition,
	revisionID string,
	now time.Time,
) ([]domain.RevisionEnvironment, []domain.RevisionArtifactFile, error) {
	sourceDir := filepath.Dir(definition.SourcePath)
	values := make(map[string]domain.RevisionEnvironment)
	var artifactFiles []domain.RevisionArtifactFile
	for _, envFile := range definition.Environment.Files {
		relative := filepath.Clean(strings.TrimSpace(envFile))
		if relative == "" || relative == "." || filepath.IsAbs(relative) || strings.HasPrefix(relative, "..") {
			return nil, nil, fmt.Errorf("%w: environment.files path %q must stay inside the definition folder", domain.ErrInvalidDefinition, envFile)
		}
		sourcePath := filepath.Join(sourceDir, relative)
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: read environment file %q: %v", domain.ErrInvalidDefinition, envFile, err)
		}
		for _, entry := range parseSimpleEnvFile(string(body)) {
			values[entry.Key] = domain.RevisionEnvironment{
				AgentName:            definition.Name,
				RevisionID:           revisionID,
				Key:                  entry.Key,
				Value:                entry.Value,
				Source:               domain.RevisionEnvironmentSourceEnvFile,
				SourcePath:           sourcePath,
				ArtifactRelativePath: filepath.ToSlash(relative),
				Masked:               true,
				CreatedAt:            now,
			}
		}
		sum := sha256.Sum256(body)
		artifactFiles = append(artifactFiles, domain.RevisionArtifactFile{
			AgentName:            definition.Name,
			RevisionID:           revisionID,
			ArtifactRelativePath: filepath.ToSlash(relative),
			SourcePath:           sourcePath,
			SHA256:               hex.EncodeToString(sum[:]),
			SizeBytes:            int64(len(body)),
			CopiedAt:             now,
		})
	}
	keys := make([]string, 0, len(definition.Environment.Variables))
	for key := range definition.Environment.Variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values[key] = domain.RevisionEnvironment{
			AgentName:  definition.Name,
			RevisionID: revisionID,
			Key:        key,
			Value:      definition.Environment.Variables[key],
			Source:     domain.RevisionEnvironmentSourceLiteral,
			Masked:     true,
			CreatedAt:  now,
		}
	}

	environment := make([]domain.RevisionEnvironment, 0, len(values))
	keys = keys[:0]
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		environment = append(environment, values[key])
	}
	sort.Slice(artifactFiles, func(i, j int) bool {
		return artifactFiles[i].ArtifactRelativePath < artifactFiles[j].ArtifactRelativePath
	})

	return environment, artifactFiles, nil
}

type simpleEnvEntry struct {
	Key   string
	Value string
}

func parseSimpleEnvFile(body string) []simpleEnvEntry {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	entries := make([]simpleEnvEntry, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		entries = append(entries, simpleEnvEntry{
			Key:   strings.TrimSpace(key),
			Value: strings.Trim(strings.TrimSpace(value), `"'`),
		})
	}

	return entries
}
