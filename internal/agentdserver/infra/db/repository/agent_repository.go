package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
)

var errAgentRepositoryNilDB = errors.New("agent repository requires a non-nil db")

type AgentRepository struct {
	db *sql.DB
}

var _ app.AgentRepository = (*AgentRepository)(nil)
var _ app.AgentRevisionRepository = (*AgentRepository)(nil)

func NewAgentRepository(database *db.DB) (*AgentRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errAgentRepositoryNilDB
	}

	return &AgentRepository{db: database.DB}, nil
}

func (r *AgentRepository) Save(
	ctx context.Context,
	agent domain.Agent,
	tools []domain.ToolPermission,
	mcpServers []domain.ToolPermission,
) error {
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = now
	}
	if agent.AppliedAt.IsZero() {
		agent.AppliedAt = now
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save agent tx: %w", err)
	}
	if err := r.upsertAgent(ctx, tx, agent); err != nil {
		_ = tx.Rollback()

		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_tools WHERE agent_name = ?`, agent.Name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete agent tools: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_mcp_servers WHERE agent_name = ?`, agent.Name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete agent mcp servers: %w", err)
	}
	for _, tool := range tools {
		tool.AgentName = agent.Name
		if err := insertTool(ctx, tx, tool, now); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	for _, server := range mcpServers {
		server.AgentName = agent.Name
		if err := insertMCPServer(ctx, tx, server, now); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save agent tx: %w", err)
	}

	return nil
}

func (r *AgentRepository) FindByName(ctx context.Context, name string) (domain.Agent, error) {
	const query = `SELECT name, revision, definition_source_path, definition_markdown,
	       prompt, enabled, vendor_name, vendor_model, schedule_type,
	       schedule_expression, next_run_at, status, last_run_id, last_error,
	       created_at, updated_at, applied_at, contract_input_schema_raw,
	       contract_output_schema_raw, contract_input_schema_digest,
	       contract_output_schema_digest
	       FROM agents WHERE name = ?`

	agent, err := scanAgent(r.db.QueryRowContext(ctx, query, name))
	if err != nil {
		return domain.Agent{}, err
	}
	if err := r.loadPolicies(ctx, &agent); err != nil {
		return domain.Agent{}, err
	}

	return agent, nil
}

func (r *AgentRepository) List(ctx context.Context) ([]domain.Agent, error) {
	const query = `SELECT name, revision, definition_source_path, definition_markdown,
	       prompt, enabled, vendor_name, vendor_model, schedule_type,
	       schedule_expression, next_run_at, status, last_run_id, last_error,
	       created_at, updated_at, applied_at, contract_input_schema_raw,
	       contract_output_schema_raw, contract_input_schema_digest,
	       contract_output_schema_digest
	       FROM agents ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var agents []domain.Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}
	for i := range agents {
		if err := r.loadPolicies(ctx, &agents[i]); err != nil {
			return nil, err
		}
	}

	return agents, nil
}

func (r *AgentRepository) SaveRevision(ctx context.Context, revision domain.AgentRevision) error {
	now := time.Now().UTC()
	if revision.CreatedAt.IsZero() {
		revision.CreatedAt = now
	}
	if revision.Status == "" {
		revision.Status = domain.AgentRevisionStatusPending
	}
	if strings.TrimSpace(revision.EnvironmentJSON) == "" {
		revision.EnvironmentJSON = "[]"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save revision tx: %w", err)
	}
	if err := upsertRevision(ctx, tx, revision); err != nil {
		_ = tx.Rollback()

		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM agent_revision_tools WHERE agent_name = ? AND revision_id = ?`,
		revision.AgentName,
		revision.RevisionID,
	); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete revision tools: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM agent_revision_artifact_files WHERE agent_name = ? AND revision_id = ?`,
		revision.AgentName,
		revision.RevisionID,
	); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete revision artifact files: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM agent_revision_environment WHERE agent_name = ? AND revision_id = ?`,
		revision.AgentName,
		revision.RevisionID,
	); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete revision environment: %w", err)
	}
	for _, tool := range revision.Tools {
		tool.AgentName = revision.AgentName
		tool.RevisionID = revision.RevisionID
		if tool.CreatedAt.IsZero() {
			tool.CreatedAt = revision.CreatedAt
		}
		if err := insertRevisionTool(ctx, tx, tool); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	for _, file := range revision.ArtifactFiles {
		file.AgentName = revision.AgentName
		file.RevisionID = revision.RevisionID
		if file.CopiedAt.IsZero() {
			file.CopiedAt = revision.CreatedAt
		}
		if err := insertRevisionArtifactFile(ctx, tx, file); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	for _, env := range revision.Environment {
		env.AgentName = revision.AgentName
		env.RevisionID = revision.RevisionID
		if env.CreatedAt.IsZero() {
			env.CreatedAt = revision.CreatedAt
		}
		if err := insertRevisionEnvironment(ctx, tx, env); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	if revision.Status == domain.AgentRevisionStatusFinalized {
		if err := updateAgentLatestRevision(ctx, tx, revision); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save revision tx: %w", err)
	}

	return nil
}

func (r *AgentRepository) ListRevisions(ctx context.Context, agentName string) ([]domain.AgentRevision, error) {
	const query = `SELECT agent_name, revision_id, content_digest, source_path, artifact_path,
	       environment_json, prompt, vendor_name, vendor_model, schedule_type,
	       schedule_expression, status, created_at, finalized_at, error_message,
	       contract_input_schema_raw, contract_output_schema_raw,
	       contract_input_schema_digest, contract_output_schema_digest, contract_digest
	       FROM agent_revisions WHERE agent_name = ? ORDER BY created_at DESC, revision_id DESC`

	rows, err := r.db.QueryContext(ctx, query, agentName)
	if err != nil {
		return nil, fmt.Errorf("query agent revisions: %w", err)
	}

	var revisions []domain.AgentRevision
	for rows.Next() {
		revision, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent revisions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close agent revisions: %w", err)
	}
	for i := range revisions {
		if err := r.loadRevisionDetails(ctx, &revisions[i]); err != nil {
			return nil, err
		}
	}
	latestID := latestFinalizedRevisionID(revisions)
	for i := range revisions {
		revisions[i].IsLatestFinalized = revisions[i].RevisionID == latestID
	}

	return revisions, nil
}

func (r *AgentRepository) FindRevisionByID(
	ctx context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, error) {
	const query = `SELECT agent_name, revision_id, content_digest, source_path, artifact_path,
	       environment_json, prompt, vendor_name, vendor_model, schedule_type,
	       schedule_expression, status, created_at, finalized_at, error_message,
	       contract_input_schema_raw, contract_output_schema_raw,
	       contract_input_schema_digest, contract_output_schema_digest, contract_digest
	       FROM agent_revisions WHERE agent_name = ? AND revision_id = ?`

	revision, err := scanRevision(r.db.QueryRowContext(ctx, query, agentName, revisionID))
	if err != nil {
		return domain.AgentRevision{}, err
	}
	if err := r.loadRevisionDetails(ctx, &revision); err != nil {
		return domain.AgentRevision{}, err
	}
	if latest, err := r.FindLatestFinalizedRevision(ctx, agentName); err == nil {
		revision.IsLatestFinalized = revision.RevisionID == latest.RevisionID
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.AgentRevision{}, err
	}

	return revision, nil
}

func (r *AgentRepository) FindRevisionByDigest(
	ctx context.Context,
	agentName string,
	contentDigest string,
) (domain.AgentRevision, error) {
	const query = `SELECT agent_name, revision_id, content_digest, source_path, artifact_path,
	       environment_json, prompt, vendor_name, vendor_model, schedule_type,
	       schedule_expression, status, created_at, finalized_at, error_message,
	       contract_input_schema_raw, contract_output_schema_raw,
	       contract_input_schema_digest, contract_output_schema_digest, contract_digest
	       FROM agent_revisions WHERE agent_name = ? AND content_digest = ?`

	revision, err := scanRevision(r.db.QueryRowContext(ctx, query, agentName, contentDigest))
	if err != nil {
		return domain.AgentRevision{}, err
	}
	if err := r.loadRevisionDetails(ctx, &revision); err != nil {
		return domain.AgentRevision{}, err
	}
	if latest, err := r.FindLatestFinalizedRevision(ctx, agentName); err == nil {
		revision.IsLatestFinalized = revision.RevisionID == latest.RevisionID
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.AgentRevision{}, err
	}

	return revision, nil
}

func (r *AgentRepository) FindLatestFinalizedRevision(
	ctx context.Context,
	agentName string,
) (domain.AgentRevision, error) {
	const query = `SELECT agent_name, revision_id, content_digest, source_path, artifact_path,
	       environment_json, prompt, vendor_name, vendor_model, schedule_type,
	       schedule_expression, status, created_at, finalized_at, error_message,
	       contract_input_schema_raw, contract_output_schema_raw,
	       contract_input_schema_digest, contract_output_schema_digest, contract_digest
	       FROM agent_revisions
	       WHERE agent_name = ? AND status = ?
	       ORDER BY created_at DESC, revision_id DESC
	       LIMIT 1`

	revision, err := scanRevision(r.db.QueryRowContext(
		ctx,
		query,
		agentName,
		domain.AgentRevisionStatusFinalized,
	))
	if err != nil {
		return domain.AgentRevision{}, err
	}
	if err := r.loadRevisionDetails(ctx, &revision); err != nil {
		return domain.AgentRevision{}, err
	}
	revision.IsLatestFinalized = true

	return revision, nil
}

func (r *AgentRepository) MarkRevisionCorrupt(
	ctx context.Context,
	agentName string,
	revisionID string,
	errorMessage string,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE agent_revisions
		 SET status = ?, error_message = ?
		 WHERE agent_name = ? AND revision_id = ?`,
		domain.AgentRevisionStatusCorrupt,
		nullString(errorMessage),
		agentName,
		revisionID,
	)
	if err != nil {
		return fmt.Errorf("mark revision corrupt: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revision corrupt rows affected: %w", err)
	}
	if count == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *AgentRepository) loadPolicies(ctx context.Context, agent *domain.Agent) error {
	tools, err := r.listTools(ctx, agent.Name)
	if err != nil {
		return err
	}
	mcpServers, err := r.listMCPServers(ctx, agent.Name)
	if err != nil {
		return err
	}
	agent.Tools = tools
	agent.MCPServers = mcpServers

	return nil
}

func (r *AgentRepository) listTools(ctx context.Context, agentName string) ([]domain.ToolPermission, error) {
	const query = `SELECT agent_name, name, kind, command, args_json, env_json,
	       read_paths_json, write_paths_json, network_allow_json
	       FROM agent_tools WHERE agent_name = ? ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, agentName)
	if err != nil {
		return nil, fmt.Errorf("query agent tools: %w", err)
	}
	defer rows.Close()

	var tools []domain.ToolPermission
	for rows.Next() {
		var tool domain.ToolPermission
		var command sql.NullString
		var args, env, readPaths, writePaths, networkAllow string
		if err := rows.Scan(
			&tool.AgentName,
			&tool.Name,
			&tool.Kind,
			&command,
			&args,
			&env,
			&readPaths,
			&writePaths,
			&networkAllow,
		); err != nil {
			return nil, fmt.Errorf("scan agent tool: %w", err)
		}
		tool.Command = command.String
		var err error
		if tool.Args, err = unmarshalList(args); err != nil {
			return nil, err
		}
		if tool.Env, err = unmarshalList(env); err != nil {
			return nil, err
		}
		if tool.ReadPaths, err = unmarshalList(readPaths); err != nil {
			return nil, err
		}
		if tool.WritePaths, err = unmarshalList(writePaths); err != nil {
			return nil, err
		}
		if tool.NetworkAllow, err = unmarshalList(networkAllow); err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent tools: %w", err)
	}

	return tools, nil
}

func (r *AgentRepository) listMCPServers(ctx context.Context, agentName string) ([]domain.ToolPermission, error) {
	const query = `SELECT agent_name, name, command, args_json, env_json
	       FROM agent_mcp_servers WHERE agent_name = ? ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, agentName)
	if err != nil {
		return nil, fmt.Errorf("query agent mcp servers: %w", err)
	}
	defer rows.Close()

	var servers []domain.ToolPermission
	for rows.Next() {
		var server domain.ToolPermission
		var args, env string
		if err := rows.Scan(
			&server.AgentName,
			&server.Name,
			&server.Command,
			&args,
			&env,
		); err != nil {
			return nil, fmt.Errorf("scan agent mcp server: %w", err)
		}
		server.Kind = domain.ToolKindMCPServer
		var err error
		if server.Args, err = unmarshalList(args); err != nil {
			return nil, err
		}
		if server.Env, err = unmarshalList(env); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent mcp servers: %w", err)
	}

	return servers, nil
}

func (r *AgentRepository) upsertAgent(
	ctx context.Context,
	tx *sql.Tx,
	agent domain.Agent,
) error {
	const query = `INSERT INTO agents (
	           name, revision, definition_source_path, definition_markdown,
	           prompt, enabled, vendor_name, vendor_model, schedule_type,
	           schedule_expression, next_run_at, status, last_run_id, last_error,
	           created_at, updated_at, applied_at, contract_input_schema_raw,
	           contract_output_schema_raw, contract_input_schema_digest,
	           contract_output_schema_digest
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	       ON CONFLICT(name) DO UPDATE SET
	           revision = excluded.revision,
	           definition_source_path = excluded.definition_source_path,
	           definition_markdown = excluded.definition_markdown,
	           prompt = excluded.prompt,
	           enabled = excluded.enabled,
	           vendor_name = excluded.vendor_name,
	           vendor_model = excluded.vendor_model,
	           schedule_type = excluded.schedule_type,
	           schedule_expression = excluded.schedule_expression,
	           next_run_at = excluded.next_run_at,
	           status = excluded.status,
	           last_run_id = excluded.last_run_id,
	           last_error = excluded.last_error,
	           updated_at = excluded.updated_at,
	           applied_at = excluded.applied_at,
	           contract_input_schema_raw = excluded.contract_input_schema_raw,
	           contract_output_schema_raw = excluded.contract_output_schema_raw,
	           contract_input_schema_digest = excluded.contract_input_schema_digest,
	           contract_output_schema_digest = excluded.contract_output_schema_digest`

	if _, err := tx.ExecContext(ctx, query,
		agent.Name,
		agent.Revision,
		agent.DefinitionSource,
		agent.DefinitionMarkdown,
		agent.Prompt,
		boolToInt(agent.Enabled),
		agent.Vendor.Name,
		agent.Vendor.Model,
		agent.Schedule.Type,
		nullString(agent.Schedule.Expression),
		nullTime(agent.NextRunAt),
		agent.Status,
		nullString(agent.LastRunID),
		nullString(agent.LastError),
		formatTime(agent.CreatedAt),
		formatTime(agent.UpdatedAt),
		formatTime(agent.AppliedAt),
		contractInputSchemaRaw(agent.Contract),
		contractOutputSchemaRaw(agent.Contract),
		contractInputSchemaDigest(agent.Contract),
		contractOutputSchemaDigest(agent.Contract),
	); err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}

	return nil
}

func insertTool(ctx context.Context, tx *sql.Tx, tool domain.ToolPermission, now time.Time) error {
	args, err := marshalList(tool.Args)
	if err != nil {
		return err
	}
	env, err := marshalList(tool.Env)
	if err != nil {
		return err
	}
	readPaths, err := marshalList(tool.ReadPaths)
	if err != nil {
		return err
	}
	writePaths, err := marshalList(tool.WritePaths)
	if err != nil {
		return err
	}
	networkAllow, err := marshalList(tool.NetworkAllow)
	if err != nil {
		return err
	}

	const query = `INSERT INTO agent_tools (
	           agent_name, name, kind, command, args_json, env_json,
	           read_paths_json, write_paths_json, network_allow_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		tool.AgentName,
		tool.Name,
		tool.Kind,
		nullString(tool.Command),
		args,
		env,
		readPaths,
		writePaths,
		networkAllow,
		formatTime(now),
	); err != nil {
		return fmt.Errorf("insert agent tool %q: %w", tool.Name, err)
	}

	return nil
}

func insertMCPServer(
	ctx context.Context,
	tx *sql.Tx,
	server domain.ToolPermission,
	now time.Time,
) error {
	args, err := marshalList(server.Args)
	if err != nil {
		return err
	}
	env, err := marshalList(server.Env)
	if err != nil {
		return err
	}

	const query = `INSERT INTO agent_mcp_servers (
	           agent_name, name, command, args_json, env_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		server.AgentName,
		server.Name,
		server.Command,
		args,
		env,
		formatTime(now),
	); err != nil {
		return fmt.Errorf("insert agent mcp server %q: %w", server.Name, err)
	}

	return nil
}

func upsertRevision(ctx context.Context, tx *sql.Tx, revision domain.AgentRevision) error {
	const query = `INSERT INTO agent_revisions (
	           agent_name, revision_id, content_digest, source_path, artifact_path,
	           environment_json, prompt, vendor_name, vendor_model, schedule_type,
	           schedule_expression, status, created_at, finalized_at, error_message,
	           contract_input_schema_raw, contract_output_schema_raw,
	           contract_input_schema_digest, contract_output_schema_digest, contract_digest
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	       ON CONFLICT(agent_name, revision_id) DO UPDATE SET
	           content_digest = excluded.content_digest,
	           source_path = excluded.source_path,
	           artifact_path = excluded.artifact_path,
	           environment_json = excluded.environment_json,
	           prompt = excluded.prompt,
	           vendor_name = excluded.vendor_name,
	           vendor_model = excluded.vendor_model,
	           schedule_type = excluded.schedule_type,
	           schedule_expression = excluded.schedule_expression,
	           status = excluded.status,
	           finalized_at = excluded.finalized_at,
	           error_message = excluded.error_message,
	           contract_input_schema_raw = excluded.contract_input_schema_raw,
	           contract_output_schema_raw = excluded.contract_output_schema_raw,
	           contract_input_schema_digest = excluded.contract_input_schema_digest,
	           contract_output_schema_digest = excluded.contract_output_schema_digest,
	           contract_digest = excluded.contract_digest`

	if _, err := tx.ExecContext(ctx, query,
		revision.AgentName,
		revision.RevisionID,
		revision.ContentDigest,
		revision.SourcePath,
		revision.ArtifactPath,
		revision.EnvironmentJSON,
		revision.Prompt,
		revision.Vendor.Name,
		revision.Vendor.Model,
		revision.Schedule.Type,
		nullString(revision.Schedule.Expression),
		revision.Status,
		formatTime(revision.CreatedAt),
		nullTime(revision.FinalizedAt),
		nullString(revision.ErrorMessage),
		nullString(revision.ContractInputSchemaRaw),
		nullString(revision.ContractOutputSchemaRaw),
		nullString(revision.ContractInputSchemaDigest),
		nullString(revision.ContractOutputSchemaDigest),
		nullString(revision.ContractDigest),
	); err != nil {
		return fmt.Errorf("upsert agent revision %q: %w", revision.RevisionID, err)
	}

	return nil
}

func insertRevisionTool(ctx context.Context, tx *sql.Tx, tool domain.RevisionTool) error {
	args, err := marshalList(tool.Args)
	if err != nil {
		return err
	}
	env, err := marshalList(tool.Env)
	if err != nil {
		return err
	}
	readPaths, err := marshalList(tool.ReadPaths)
	if err != nil {
		return err
	}
	writePaths, err := marshalList(tool.WritePaths)
	if err != nil {
		return err
	}
	networkAllow, err := marshalList(tool.NetworkAllow)
	if err != nil {
		return err
	}
	copiedFiles, err := marshalList(tool.CopiedFiles)
	if err != nil {
		return err
	}

	const query = `INSERT INTO agent_revision_tools (
	           agent_name, revision_id, name, kind, original_command, rewritten_command,
	           host_command, args_json, env_json, timeout_seconds, read_paths_json,
	           write_paths_json, network_allow_json, copied_files_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		tool.AgentName,
		tool.RevisionID,
		tool.Name,
		tool.Kind,
		nullString(tool.OriginalCommand),
		nullString(tool.RewrittenCommand),
		nullString(tool.HostCommand),
		args,
		env,
		timeoutSeconds(tool.Timeout),
		readPaths,
		writePaths,
		networkAllow,
		copiedFiles,
		formatTime(tool.CreatedAt),
	); err != nil {
		return fmt.Errorf("insert revision tool %q: %w", tool.Name, err)
	}

	return nil
}

func insertRevisionArtifactFile(
	ctx context.Context,
	tx *sql.Tx,
	file domain.RevisionArtifactFile,
) error {
	const query = `INSERT INTO agent_revision_artifact_files (
	           agent_name, revision_id, artifact_relative_path, source_path,
	           sha256, mode, size_bytes, copied_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		file.AgentName,
		file.RevisionID,
		file.ArtifactRelativePath,
		file.SourcePath,
		file.SHA256,
		file.Mode,
		file.SizeBytes,
		formatTime(file.CopiedAt),
	); err != nil {
		return fmt.Errorf("insert revision artifact file %q: %w", file.ArtifactRelativePath, err)
	}

	return nil
}

func insertRevisionEnvironment(
	ctx context.Context,
	tx *sql.Tx,
	env domain.RevisionEnvironment,
) error {
	const query = `INSERT INTO agent_revision_environment (
	           agent_name, revision_id, key, value, source, source_path,
	           artifact_relative_path, masked, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		env.AgentName,
		env.RevisionID,
		env.Key,
		env.Value,
		env.Source,
		nullString(env.SourcePath),
		nullString(env.ArtifactRelativePath),
		boolToInt(env.Masked),
		formatTime(env.CreatedAt),
	); err != nil {
		return fmt.Errorf("insert revision environment %q: %w", env.Key, err)
	}

	return nil
}

func updateAgentLatestRevision(
	ctx context.Context,
	tx *sql.Tx,
	revision domain.AgentRevision,
) error {
	result, err := tx.ExecContext(
		ctx,
		`UPDATE agents SET revision = ?, updated_at = ? WHERE name = ?`,
		revision.RevisionID,
		formatTime(revision.CreatedAt),
		revision.AgentName,
	)
	if err != nil {
		return fmt.Errorf("update agent latest revision: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("agent latest revision rows affected: %w", err)
	}
	if count == 0 {
		return domain.ErrNotFound
	}

	return nil
}

type agentScanner interface {
	Scan(dest ...any) error
}

func scanAgent(scanner agentScanner) (domain.Agent, error) {
	var (
		agent                                               domain.Agent
		enabled                                             int
		scheduleExpression, nextRunAt, lastRunID, lastError sql.NullString
		contractInputRaw, contractOutputRaw                 sql.NullString
		contractInputDigest, contractOutputDigest           sql.NullString
		createdAt, updatedAt, appliedAt                     string
	)

	err := scanner.Scan(
		&agent.Name,
		&agent.Revision,
		&agent.DefinitionSource,
		&agent.DefinitionMarkdown,
		&agent.Prompt,
		&enabled,
		&agent.Vendor.Name,
		&agent.Vendor.Model,
		&agent.Schedule.Type,
		&scheduleExpression,
		&nextRunAt,
		&agent.Status,
		&lastRunID,
		&lastError,
		&createdAt,
		&updatedAt,
		&appliedAt,
		&contractInputRaw,
		&contractOutputRaw,
		&contractInputDigest,
		&contractOutputDigest,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Agent{}, domain.ErrNotFound
		}

		return domain.Agent{}, fmt.Errorf("scan agent: %w", err)
	}

	agent.Enabled = enabled == 1
	agent.Schedule.Expression = scheduleExpression.String
	agent.LastRunID = lastRunID.String
	agent.LastError = lastError.String
	if nextRunAt.Valid {
		parsed, err := parseTime(nextRunAt.String)
		if err != nil {
			return domain.Agent{}, err
		}
		agent.NextRunAt = &parsed
	}
	if agent.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Agent{}, err
	}
	if agent.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Agent{}, err
	}
	if agent.AppliedAt, err = parseTime(appliedAt); err != nil {
		return domain.Agent{}, err
	}
	agent.Contract = scanContract(
		contractInputRaw,
		contractOutputRaw,
		contractInputDigest,
		contractOutputDigest,
	)

	return agent, nil
}

func (r *AgentRepository) loadRevisionDetails(ctx context.Context, revision *domain.AgentRevision) error {
	tools, err := r.listRevisionTools(ctx, revision.AgentName, revision.RevisionID)
	if err != nil {
		return err
	}
	files, err := r.listRevisionArtifactFiles(ctx, revision.AgentName, revision.RevisionID)
	if err != nil {
		return err
	}
	env, err := r.listRevisionEnvironment(ctx, revision.AgentName, revision.RevisionID)
	if err != nil {
		return err
	}
	revision.Tools = tools
	revision.ArtifactFiles = files
	revision.Environment = env

	return nil
}

func (r *AgentRepository) listRevisionTools(
	ctx context.Context,
	agentName string,
	revisionID string,
) ([]domain.RevisionTool, error) {
	const query = `SELECT agent_name, revision_id, name, kind, original_command,
	       rewritten_command, host_command, args_json, env_json, timeout_seconds,
	       read_paths_json, write_paths_json, network_allow_json, copied_files_json,
	       created_at
	       FROM agent_revision_tools
	       WHERE agent_name = ? AND revision_id = ?
	       ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, agentName, revisionID)
	if err != nil {
		return nil, fmt.Errorf("query revision tools: %w", err)
	}
	defer rows.Close()

	var tools []domain.RevisionTool
	for rows.Next() {
		var (
			tool                                                        domain.RevisionTool
			originalCommand, rewrittenCommand, hostCommand              sql.NullString
			args, env, readPaths, writePaths, networkAllow, copiedFiles string
			timeout                                                     int64
			createdAt                                                   string
		)
		if err := rows.Scan(
			&tool.AgentName,
			&tool.RevisionID,
			&tool.Name,
			&tool.Kind,
			&originalCommand,
			&rewrittenCommand,
			&hostCommand,
			&args,
			&env,
			&timeout,
			&readPaths,
			&writePaths,
			&networkAllow,
			&copiedFiles,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan revision tool: %w", err)
		}
		tool.OriginalCommand = originalCommand.String
		tool.RewrittenCommand = rewrittenCommand.String
		tool.HostCommand = hostCommand.String
		tool.Timeout = formatTimeoutSeconds(timeout)
		var err error
		if tool.Args, err = unmarshalList(args); err != nil {
			return nil, err
		}
		if tool.Env, err = unmarshalList(env); err != nil {
			return nil, err
		}
		if tool.ReadPaths, err = unmarshalList(readPaths); err != nil {
			return nil, err
		}
		if tool.WritePaths, err = unmarshalList(writePaths); err != nil {
			return nil, err
		}
		if tool.NetworkAllow, err = unmarshalList(networkAllow); err != nil {
			return nil, err
		}
		if tool.CopiedFiles, err = unmarshalList(copiedFiles); err != nil {
			return nil, err
		}
		if tool.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revision tools: %w", err)
	}

	return tools, nil
}

func (r *AgentRepository) listRevisionArtifactFiles(
	ctx context.Context,
	agentName string,
	revisionID string,
) ([]domain.RevisionArtifactFile, error) {
	const query = `SELECT agent_name, revision_id, artifact_relative_path, source_path,
	       sha256, mode, size_bytes, copied_at
	       FROM agent_revision_artifact_files
	       WHERE agent_name = ? AND revision_id = ?
	       ORDER BY artifact_relative_path`

	rows, err := r.db.QueryContext(ctx, query, agentName, revisionID)
	if err != nil {
		return nil, fmt.Errorf("query revision artifact files: %w", err)
	}
	defer rows.Close()

	var files []domain.RevisionArtifactFile
	for rows.Next() {
		var (
			file     domain.RevisionArtifactFile
			copiedAt string
		)
		if err := rows.Scan(
			&file.AgentName,
			&file.RevisionID,
			&file.ArtifactRelativePath,
			&file.SourcePath,
			&file.SHA256,
			&file.Mode,
			&file.SizeBytes,
			&copiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan revision artifact file: %w", err)
		}
		parsed, err := parseTime(copiedAt)
		if err != nil {
			return nil, err
		}
		file.CopiedAt = parsed
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revision artifact files: %w", err)
	}

	return files, nil
}

func (r *AgentRepository) listRevisionEnvironment(
	ctx context.Context,
	agentName string,
	revisionID string,
) ([]domain.RevisionEnvironment, error) {
	const query = `SELECT agent_name, revision_id, key, value, source, source_path,
	       artifact_relative_path, masked, created_at
	       FROM agent_revision_environment
	       WHERE agent_name = ? AND revision_id = ?
	       ORDER BY key, source`

	rows, err := r.db.QueryContext(ctx, query, agentName, revisionID)
	if err != nil {
		return nil, fmt.Errorf("query revision environment: %w", err)
	}
	defer rows.Close()

	var environment []domain.RevisionEnvironment
	for rows.Next() {
		var (
			env                          domain.RevisionEnvironment
			sourcePath, artifactRelative sql.NullString
			masked                       int
			createdAt                    string
		)
		if err := rows.Scan(
			&env.AgentName,
			&env.RevisionID,
			&env.Key,
			&env.Value,
			&env.Source,
			&sourcePath,
			&artifactRelative,
			&masked,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan revision environment: %w", err)
		}
		env.SourcePath = sourcePath.String
		env.ArtifactRelativePath = artifactRelative.String
		env.Masked = masked == 1
		parsed, err := parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		env.CreatedAt = parsed
		environment = append(environment, env)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revision environment: %w", err)
	}

	return environment, nil
}

func scanRevision(scanner agentScanner) (domain.AgentRevision, error) {
	var (
		revision                                      domain.AgentRevision
		scheduleExpression, finalizedAt, errorMessage sql.NullString
		contractInputRaw, contractOutputRaw           sql.NullString
		contractInputDigest, contractOutputDigest     sql.NullString
		contractDigest                                sql.NullString
		createdAt                                     string
	)

	err := scanner.Scan(
		&revision.AgentName,
		&revision.RevisionID,
		&revision.ContentDigest,
		&revision.SourcePath,
		&revision.ArtifactPath,
		&revision.EnvironmentJSON,
		&revision.Prompt,
		&revision.Vendor.Name,
		&revision.Vendor.Model,
		&revision.Schedule.Type,
		&scheduleExpression,
		&revision.Status,
		&createdAt,
		&finalizedAt,
		&errorMessage,
		&contractInputRaw,
		&contractOutputRaw,
		&contractInputDigest,
		&contractOutputDigest,
		&contractDigest,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AgentRevision{}, domain.ErrNotFound
		}

		return domain.AgentRevision{}, fmt.Errorf("scan agent revision: %w", err)
	}
	revision.Schedule.Expression = scheduleExpression.String
	revision.ErrorMessage = errorMessage.String
	if finalizedAt.Valid {
		parsed, err := parseTime(finalizedAt.String)
		if err != nil {
			return domain.AgentRevision{}, err
		}
		revision.FinalizedAt = &parsed
	}
	var errParse error
	if revision.CreatedAt, errParse = parseTime(createdAt); errParse != nil {
		return domain.AgentRevision{}, errParse
	}
	revision.ContractInputSchemaRaw = contractInputRaw.String
	revision.ContractOutputSchemaRaw = contractOutputRaw.String
	revision.ContractInputSchemaDigest = contractInputDigest.String
	revision.ContractOutputSchemaDigest = contractOutputDigest.String
	revision.ContractDigest = contractDigest.String

	return revision, nil
}

func scanContract(inputRaw, outputRaw, inputDigest, outputDigest sql.NullString) *domain.AgentContract {
	if !inputRaw.Valid && !outputRaw.Valid && !inputDigest.Valid && !outputDigest.Valid {
		return nil
	}

	return &domain.AgentContract{
		InputSchemaRaw:     inputRaw.String,
		OutputSchemaRaw:    outputRaw.String,
		InputSchemaDigest:  inputDigest.String,
		OutputSchemaDigest: outputDigest.String,
	}
}

func marshalList(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	body, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal string list: %w", err)
	}

	return string(body), nil
}

func unmarshalList(body string) ([]string, error) {
	if strings.TrimSpace(body) == "" {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(body), &values); err != nil {
		return nil, fmt.Errorf("unmarshal string list: %w", err)
	}
	if values == nil {
		return []string{}, nil
	}

	return values, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

func contractInputSchemaRaw(contract *domain.AgentContract) any {
	if contract == nil {
		return nil
	}
	return nullString(contract.InputSchemaRaw)
}

func contractOutputSchemaRaw(contract *domain.AgentContract) any {
	if contract == nil {
		return nil
	}
	return nullString(contract.OutputSchemaRaw)
}

func contractInputSchemaDigest(contract *domain.AgentContract) any {
	if contract == nil {
		return nil
	}
	return nullString(contract.InputSchemaDigest)
}

func contractOutputSchemaDigest(contract *domain.AgentContract) any {
	if contract == nil {
		return nil
	}
	return nullString(contract.OutputSchemaDigest)
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return formatTime(*value)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", value, err)
	}

	return parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func timeoutSeconds(value string) int64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}

	return int64(duration.Seconds())
}

func formatTimeoutSeconds(value int64) string {
	if value == 0 {
		return ""
	}

	return fmt.Sprintf("%ds", value)
}

func latestFinalizedRevisionID(revisions []domain.AgentRevision) string {
	for _, revision := range revisions {
		if revision.Status == domain.AgentRevisionStatusFinalized {
			return revision.RevisionID
		}
	}

	return ""
}
